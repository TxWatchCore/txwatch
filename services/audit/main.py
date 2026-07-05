import os
import json
import time
import psycopg
from kafka import KafkaConsumer
from dotenv import load_dotenv

load_dotenv()

# ── Config ───────────────────────────────────────────────────────────────────
KAFKA_BROKER = os.getenv("KAFKA_BROKER", "localhost:9092")
KAFKA_TOPIC = os.getenv("KAFKA_TOPIC", "txwatch.decisions")
KAFKA_GROUP_ID = os.getenv("KAFKA_GROUP_ID", "audit-service")
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://txwatch:txwatch@localhost:5432/txwatch")

# ── Database setup ────────────────────────────────────────────────────────────
def get_connection():
    return psycopg.connect(DATABASE_URL)

def setup_schema(conn):
    """Create tables if they don't exist."""
    with conn.cursor() as cur:
        cur.execute("""
            CREATE TABLE IF NOT EXISTS decisions (
                id SERIAL PRIMARY KEY,
                transaction_id VARCHAR(255) UNIQUE NOT NULL,
                risk_score FLOAT NOT NULL,
                decision VARCHAR(20) NOT NULL,
                features JSONB,
                shap_values JSONB,
                model_version VARCHAR(50),
                explanation_strategy VARCHAR(50),
                feature_ms FLOAT,
                inference_ms FLOAT,
                shap_ms FLOAT,
                total_ms FLOAT,
                created_at TIMESTAMPTZ DEFAULT NOW()
            );
        """)
        cur.execute("""
            CREATE TABLE IF NOT EXISTS benchmark_runs (
                id SERIAL PRIMARY KEY,
                transaction_id VARCHAR(255),
                explanation_strategy VARCHAR(50),
                feature_ms FLOAT,
                inference_ms FLOAT,
                shap_ms FLOAT,
                total_ms FLOAT,
                created_at TIMESTAMPTZ DEFAULT NOW()
            );
        """)
        conn.commit()
    print("Schema ready.")

def insert_decision(conn, event: dict):
    """Insert a decision event into the decisions table."""
    with conn.cursor() as cur:
        cur.execute("""
            INSERT INTO decisions (
                transaction_id, risk_score, decision, features,
                shap_values, model_version, explanation_strategy,
                feature_ms, inference_ms, shap_ms, total_ms
            ) VALUES (
                %(transaction_id)s, %(risk_score)s, %(decision)s,
                %(features)s, %(shap_values)s, %(model_version)s,
                %(explanation_strategy)s, %(feature_ms)s,
                %(inference_ms)s, %(shap_ms)s, %(total_ms)s
            )
            ON CONFLICT (transaction_id) DO NOTHING;
        """, {
            "transaction_id": event.get("transaction_id"),
            "risk_score": event.get("risk_score"),
            "decision": event.get("decision"),
            "features": json.dumps(event.get("features", {})),
            "shap_values": json.dumps(event.get("shap_values", {})),
            "model_version": event.get("model_version"),
            "explanation_strategy": event.get("shap_strategy"),
            "feature_ms": event.get("feature_ms"),
            "inference_ms": event.get("inference_ms"),
            "shap_ms": event.get("shap_ms"),
            "total_ms": event.get("total_ms"),
        })
        conn.commit()

def insert_benchmark_run(conn, event: dict):
    """Insert benchmark timing into benchmark_runs table."""
    with conn.cursor() as cur:
        cur.execute("""
            INSERT INTO benchmark_runs (
                transaction_id, explanation_strategy,
                feature_ms, inference_ms, shap_ms, total_ms
            ) VALUES (
                %(transaction_id)s, %(explanation_strategy)s,
                %(feature_ms)s, %(inference_ms)s,
                %(shap_ms)s, %(total_ms)s
            );
        """, {
            "transaction_id": event.get("transaction_id"),
            "explanation_strategy": event.get("shap_strategy"),
            "feature_ms": event.get("feature_ms"),
            "inference_ms": event.get("inference_ms"),
            "shap_ms": event.get("shap_ms"),
            "total_ms": event.get("total_ms"),
        })
        conn.commit()

# ── Main consumer loop ────────────────────────────────────────────────────────
def main():
    print(f"Connecting to Kafka at {KAFKA_BROKER}...")
    consumer = KafkaConsumer(
        KAFKA_TOPIC,
        bootstrap_servers=[KAFKA_BROKER],
        group_id=KAFKA_GROUP_ID,
        value_deserializer=lambda m: json.loads(m.decode("utf-8")),
        auto_offset_reset="earliest",
        enable_auto_commit=True,
    )
    print(f"Subscribed to topic: {KAFKA_TOPIC}")

    print(f"Connecting to PostgreSQL...")
    conn = get_connection()
    setup_schema(conn)
    print("Audit service ready. Consuming events...")

    for message in consumer:
        event = message.value
        txn_id = event.get("transaction_id", "unknown")

        try:
            insert_decision(conn, event)
            insert_benchmark_run(conn, event)
            print(f"[audit] Logged decision for {txn_id} — {event.get('decision')} score={event.get('risk_score'):.4f}")
        except Exception as e:
            print(f"[audit] Error logging {txn_id}: {e}")
            # Reconnect on connection errors
            try:
                conn = get_connection()
            except Exception:
                pass

if __name__ == "__main__":
    main()
