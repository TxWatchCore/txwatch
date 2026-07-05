import time
import json
import numpy as np
import pandas as pd
import requests
from pathlib import Path
from sklearn.preprocessing import LabelEncoder
from sklearn.model_selection import train_test_split

# ── Config ────────────────────────────────────────────────────────────────────
INFERENCE_URL = "http://localhost:8001/infer"
N_SAMPLES = 100
STRATEGIES = [
    "exact_tree",
    "reduced_tree_50",
    "reduced_tree_100",
    "reduced_tree_200",
    "feature_sampled",
    "async_posthoc",
    "kernel",
]

KERNEL_SAMPLES = 20  # kernel runs on fewer samples

# ── Load test data ────────────────────────────────────────────────────────────
print("Loading test data...")
DATA_DIR = Path("../../ml/data")
tx = pd.read_csv(DATA_DIR / "train_transaction.csv")
id_df = pd.read_csv(DATA_DIR / "train_identity.csv")
df = tx.merge(id_df, on="TransactionID", how="left")

drop_cols = ["isFraud", "TransactionID"]
FEATURES = [col for col in df.columns if col not in drop_cols]
df = df[FEATURES + ["isFraud"]].copy()

for col in df.columns:
    if df[col].dtype == 'object' or str(df[col].dtype) == 'str':
        le = LabelEncoder()
        df[col] = df[col].astype(str).fillna("unknown")
        df[col] = le.fit_transform(df[col])

df = df.fillna(-999)
X = df[FEATURES].astype(np.float32)
y = df["isFraud"]

_, X_test, _, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

X_bench_full = X_test.iloc[:N_SAMPLES]
X_bench_kernel = X_test.iloc[:KERNEL_SAMPLES]
print(f"Full benchmark dataset: {len(X_bench_full)} transactions")
print(f"Kernel benchmark dataset: {len(X_bench_kernel)} transactions")

# ── Helpers ───────────────────────────────────────────────────────────────────
def run_inference(features: list, transaction_id: str) -> dict:
    payload = {"transaction_id": transaction_id, "features": features}
    response = requests.post(INFERENCE_URL, json=payload, timeout=60)
    response.raise_for_status()
    return response.json()

def compute_mae(shap_a: dict, shap_b: dict) -> float:
    keys = list(shap_a.keys())
    errors = [abs(float(shap_a[k]) - float(shap_b.get(k, 0.0))) for k in keys]
    return float(np.mean(errors))

# ── Cache paths ───────────────────────────────────────────────────────────────
docs_dir = Path("../../docs")
docs_dir.mkdir(exist_ok=True)
cache_path = docs_dir / "exact_shap_cache.json"
results_path = docs_dir / "benchmark_results.json"

# Load existing results so we can resume
existing_results = {}
if results_path.exists():
    with open(results_path) as f:
        existing_results = json.load(f)
    print(f"Loaded existing results for: {list(existing_results.keys())}")

# Load exact SHAP cache
exact_shap_cache = {}
if cache_path.exists():
    with open(cache_path) as f:
        exact_shap_cache = {int(k): v for k, v in json.load(f).items()}
    print(f"Loaded exact SHAP cache: {len(exact_shap_cache)} entries")

# ── Run benchmark ─────────────────────────────────────────────────────────────
results = existing_results.copy()

for strategy in STRATEGIES:
    if strategy in results:
        print(f"\nSkipping {strategy} — already in results")
        continue

    n_samples = KERNEL_SAMPLES if strategy == "kernel" else N_SAMPLES
    X_bench = X_bench_kernel if strategy == "kernel" else X_bench_full

    print(f"\nBenchmarking strategy: {strategy} ({n_samples} samples)")
    print(f"  Set SHAP_STRATEGY={strategy} in .env, restart sidecar, then press Enter...")
    input()

    latencies = []
    shap_times = []
    maes = []

    for i, (idx, row) in enumerate(X_bench.iterrows()):
        txn_id = f"bench_{strategy}_{i}"
        features = row.tolist()
        result = run_inference(features, txn_id)

        latencies.append(result["total_ms"])
        shap_times.append(result["shap_ms"])

        if strategy == "exact_tree":
            exact_shap_cache[i] = result.get("shap_values", {})
        elif strategy != "async_posthoc" and result.get("shap_values"):
            if i in exact_shap_cache:
                mae = compute_mae(exact_shap_cache[i], result["shap_values"])
                maes.append(mae)

        if (i + 1) % 10 == 0:
            print(f"  {i + 1}/{n_samples} done...")

    # Save exact SHAP cache immediately after exact_tree
    if strategy == "exact_tree":
        with open(cache_path, "w") as f:
            json.dump({str(k): v for k, v in exact_shap_cache.items()}, f)
        print(f"  Exact SHAP cache saved: {len(exact_shap_cache)} entries")

    latencies = np.array(latencies)
    shap_times = np.array(shap_times)

    results[strategy] = {
        "n_samples": n_samples,
        "p50_total_ms": round(float(np.percentile(latencies, 50)), 3),
        "p95_total_ms": round(float(np.percentile(latencies, 95)), 3),
        "p99_total_ms": round(float(np.percentile(latencies, 99)), 3),
        "p50_shap_ms": round(float(np.percentile(shap_times, 50)), 3),
        "p95_shap_ms": round(float(np.percentile(shap_times, 95)), 3),
        "p99_shap_ms": round(float(np.percentile(shap_times, 99)), 3),
        "mean_mae": round(float(np.mean(maes)), 6) if maes else None,
    }

    # Save results after every strategy
    with open(results_path, "w") as f:
        json.dump(results, f, indent=2)

    print(f"  p50: {results[strategy]['p50_total_ms']}ms")
    print(f"  p95: {results[strategy]['p95_total_ms']}ms")
    print(f"  p99: {results[strategy]['p99_total_ms']}ms")
    print(f"  MAE: {results[strategy]['mean_mae']}")

# ── Final summary ─────────────────────────────────────────────────────────────
print("\n" + "="*60)
print("FINAL BENCHMARK RESULTS")
print("="*60)
for strategy, metrics in results.items():
    print(f"\n{strategy}:")
    for k, v in metrics.items():
        print(f"  {k}: {v}")
