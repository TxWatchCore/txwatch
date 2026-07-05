import os
import time
import httpx
import numpy as np
import redis.asyncio as aioredis

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dotenv import load_dotenv
from contextlib import asynccontextmanager
from typing import Optional

from velocity import VelocityStore

load_dotenv()

# ── Config ───────────────────────────────────────────────────────────────────
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
INFERENCE_URL = os.getenv("INFERENCE_URL", "http://localhost:8001/infer")
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8002"))

# ── State ─────────────────────────────────────────────────────────────────────
redis_client: Optional[aioredis.Redis] = None
velocity_store: Optional[VelocityStore] = None
http_client: Optional[httpx.AsyncClient] = None

# ── Lifespan ──────────────────────────────────────────────────────────────────
@asynccontextmanager
async def lifespan(app: FastAPI):
    global redis_client, velocity_store, http_client

    print(f"Connecting to Redis at {REDIS_URL}...")
    redis_client = aioredis.from_url(REDIS_URL, decode_responses=False)
    velocity_store = VelocityStore(redis_client)
    print("Redis connected.")

    http_client = httpx.AsyncClient(timeout=10.0)
    print(f"HTTP client ready. Inference URL: {INFERENCE_URL}")

    yield

    await redis_client.aclose()
    await http_client.aclose()
    print("Feature service shut down.")


app = FastAPI(
    title="TxWatch Feature Service",
    version="1.0.0",
    lifespan=lifespan,
)

# ── Schemas ───────────────────────────────────────────────────────────────────
class TransactionRequest(BaseModel):
    transaction_id: str
    user_id: str
    amount: float
    product_cd: str
    card4: str
    card6: str
    p_emaildomain: str
    r_emaildomain: str
    transaction_dt: float
    card1: float
    card2: float
    card3: float
    card5: float
    addr1: float
    addr2: float
    dist1: float
    dist2: float

class DecisionResponse(BaseModel):
    transaction_id: str
    risk_score: float
    decision: str
    shap_values: Optional[dict] = None
    model_version: str
    shap_strategy: str
    inference_ms: float
    shap_ms: float
    total_ms: float
    feature_ms: float

# ── Feature extraction ────────────────────────────────────────────────────────
CATEGORICAL_MAPS = {
    "product_cd": {"W": 0, "H": 1, "C": 2, "S": 3, "R": 4},
    "card4": {"visa": 0, "mastercard": 1, "american express": 2, "discover": 3},
    "card6": {"debit": 0, "credit": 1, "debit or credit": 2, "charge card": 3},
}

def encode_categorical(value: str, mapping: dict) -> float:
    return float(mapping.get(value.lower(), -1))

def extract_static_features(req: TransactionRequest) -> list[float]:
    """
    Extract static features from the transaction payload.
    Order must match the training feature order.
    """
    return [
        float(req.transaction_dt),
        float(req.amount),
        encode_categorical(req.product_cd, CATEGORICAL_MAPS["product_cd"]),
        float(req.card1),
        float(req.card2),
        float(req.card3),
        encode_categorical(req.card4, CATEGORICAL_MAPS["card4"]),
        float(req.card5),
        encode_categorical(req.card6, CATEGORICAL_MAPS["card6"]),
        float(req.addr1),
        float(req.addr2),
        float(req.dist1),
        float(req.dist2),
    ]

# ── Routes ────────────────────────────────────────────────────────────────────
@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/ready")
async def ready():
    if redis_client is None:
        raise HTTPException(status_code=503, detail="Redis not connected")
    return {"status": "ready"}

@app.post("/score", response_model=DecisionResponse)
async def score(req: TransactionRequest):
    total_start = time.perf_counter()

    # ── Feature extraction ────────────────────────────────────────────────────
    feature_start = time.perf_counter()

    static_features = extract_static_features(req)
    velocity_features = await velocity_store.get_velocity_features(req.user_id)

    # Append velocity features to static features
    velocity_vector = [
        float(velocity_features["velocity_count_1h"]),
        float(velocity_features["velocity_amount_1h"]),
        float(velocity_features["velocity_count_6h"]),
        float(velocity_features["velocity_amount_6h"]),
        float(velocity_features["velocity_count_24h"]),
        float(velocity_features["velocity_amount_24h"]),
    ]

    # Pad remaining features to match model input size (432)
    base_features = static_features + velocity_vector
    padding = [0.0] * (432 - len(base_features))
    full_feature_vector = base_features + padding

    feature_ms = (time.perf_counter() - feature_start) * 1000

    # Record transaction in velocity store
    await velocity_store.record_transaction(
        user_id=req.user_id,
        transaction_id=req.transaction_id,
        amount=req.amount,
    )

    # ── Call inference sidecar ────────────────────────────────────────────────
    payload = {
        "transaction_id": req.transaction_id,
        "features": full_feature_vector,
    }

    try:
        response = await http_client.post(INFERENCE_URL, json=payload)
        response.raise_for_status()
        result = response.json()
    except httpx.HTTPError as e:
        raise HTTPException(status_code=502, detail=f"Inference sidecar error: {e}")

    total_ms = (time.perf_counter() - total_start) * 1000

    return DecisionResponse(
        transaction_id=result["transaction_id"],
        risk_score=result["risk_score"],
        decision=result["decision"],
        shap_values=result.get("shap_values"),
        model_version=result["model_version"],
        shap_strategy=result["shap_strategy"],
        inference_ms=result["inference_ms"],
        shap_ms=result["shap_ms"],
        total_ms=round(total_ms, 3),
        feature_ms=round(feature_ms, 3),
    )
