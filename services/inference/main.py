import os
import time
import numpy as np
import onnxruntime as rt

import pickle
import lightgbm as lgb

from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
from dotenv import load_dotenv
from contextlib import asynccontextmanager
from typing import Optional

from shap_strategies import load_strategy

load_dotenv()

# ── Config ───────────────────────────────────────────────────────────────────
MODEL_PATH = os.getenv("MODEL_PATH", "../../ml/models/txwatch_v2.onnx")
LGBM_PATH = os.getenv("LGBM_PATH", "../../ml/models/txwatch_v2_lgbm.pkl")
MODEL_VERSION = os.getenv("MODEL_VERSION", "v2")
THRESHOLD = float(os.getenv("THRESHOLD", "0.83"))
SHAP_STRATEGY = os.getenv("SHAP_STRATEGY", "exact_tree")
BACKGROUND_PATH = os.getenv("BACKGROUND_PATH", "../../ml/models/background.npy")

# ── Model state ──────────────────────────────────────────────────────────────
model_session: Optional[rt.InferenceSession] = None
model_input_name: Optional[str] = None
lgbm_model = None
shap_strategy = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    global model_session, model_input_name, lgbm_model, shap_strategy

    # Load ONNX model
    print(f"Loading ONNX model from {MODEL_PATH}...")
    model_session = rt.InferenceSession(MODEL_PATH)
    model_input_name = model_session.get_inputs()[0].name
    print(f"Model loaded. Input: '{model_input_name}', Version: {MODEL_VERSION}")

    # Load LightGBM model for SHAP
    print(f"Loading LightGBM model from {LGBM_PATH}...")
    with open(LGBM_PATH, "rb") as f:
        lgbm_model = pickle.load(f)
    print("LightGBM model loaded.")

    # Load background dataset
    print(f"Loading background data from {BACKGROUND_PATH}...")
    background_data = np.load(BACKGROUND_PATH)
    print(f"Background data loaded. Shape: {background_data.shape}")

    # Define predict function for KernelSHAP
    def predict_fn(X):
        preds = model_session.run(None, {model_input_name: X.astype(np.float32)})
        return np.array([p[1] for p in preds[1]])

    # Load SHAP strategy
    print(f"Loading SHAP strategy: {SHAP_STRATEGY}...")
    shap_strategy = load_strategy(
        SHAP_STRATEGY,
        model=lgbm_model,
        background_data=background_data,
        predict_fn=predict_fn,
    )
    print(f"SHAP strategy '{SHAP_STRATEGY}' ready.")
    yield
    print("Shutting down inference sidecar.")


app = FastAPI(
    title="TxWatch Inference Sidecar",
    version="1.0.0",
    lifespan=lifespan,
)

# ── Schemas ───────────────────────────────────────────────────────────────────
class InferenceRequest(BaseModel):
    transaction_id: str
    features: list[float]

class InferenceResponse(BaseModel):
    model_config = {"protected_namespaces": ()}

    transaction_id: str
    risk_score: float
    decision: str
    shap_values: Optional[dict] = None
    model_version: str
    shap_strategy: str
    inference_ms: float
    shap_ms: float
    total_ms: float

# ── Routes ────────────────────────────────────────────────────────────────────
@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/ready")
def ready():
    if model_session is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    return {"status": "ready", "model_version": MODEL_VERSION, "shap_strategy": SHAP_STRATEGY}

@app.post("/infer", response_model=InferenceResponse)
async def infer(req: InferenceRequest, background_tasks: BackgroundTasks):
    if model_session is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    total_start = time.perf_counter()

    # ── Inference ────────────────────────────────────────────────────────────
    infer_start = time.perf_counter()
    features = np.array([req.features], dtype=np.float32)
    preds = model_session.run(None, {model_input_name: features})
    risk_score = float([p[1] for p in preds[1]][0])
    inference_ms = (time.perf_counter() - infer_start) * 1000

    # ── Decision ─────────────────────────────────────────────────────────────
    if risk_score >= THRESHOLD:
        decision = "BLOCK"
    elif risk_score >= 0.5:
        decision = "FLAG"
    else:
        decision = "APPROVE"

    # ── SHAP ─────────────────────────────────────────────────────────────────
    shap_values, shap_ms = shap_strategy.compute(features)

    # For async strategy — dispatch background computation
    if SHAP_STRATEGY == "async_posthoc":
        background_tasks.add_task(
            _async_shap_background,
            req.transaction_id,
            features
        )
        shap_values = None

    total_ms = (time.perf_counter() - total_start) * 1000

    return InferenceResponse(
        transaction_id=req.transaction_id,
        risk_score=round(risk_score, 6),
        decision=decision,
        shap_values=shap_values,
        model_version=MODEL_VERSION,
        shap_strategy=SHAP_STRATEGY,
        inference_ms=round(inference_ms, 3),
        shap_ms=round(shap_ms, 3),
        total_ms=round(total_ms, 3),
    )


async def _async_shap_background(transaction_id: str, features: np.ndarray):
    """Background task for async post-hoc SHAP computation."""
    print(f"[async_shap] Computing SHAP for {transaction_id}...")
    shap_vals, ms = shap_strategy.compute_background(features)
    print(f"[async_shap] Done for {transaction_id} in {ms:.1f}ms")
    # In full system this would publish to Kafka
