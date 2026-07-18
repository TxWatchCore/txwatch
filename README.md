# TxWatch

Real-time card-transaction fraud detection. Trains a gradient-boosted model on the IEEE-CIS Fraud Detection dataset, exports it to ONNX, and (once the service layer is built) exposes scoring, feature lookup, and audit trail over HTTP.

**Status:** Python rewrite in progress. The ML pipeline is working end-to-end and has produced two trained models. The five HTTP services are scaffolded but not yet implemented.

## Layout

```
txwatch/
├── ml/                     # training pipeline (working)
│   ├── training/           # train / evaluate scripts
│   ├── data/               # raw CSVs, gitignored
│   ├── models/             # trained ONNX + encoder artifacts
│   └── notebooks/          # feature exploration
├── services/               # runtime services (scaffolded)
│   ├── inference/          # ONNX scoring + SHAP explainability
│   ├── feature/            # Redis-backed velocity features
│   ├── decision/           # threshold policy → allow/review/block
│   ├── gateway/            # public entrypoint, idempotency
│   └── audit/              # Kafka consumer → PostgreSQL trail
├── docs/adr/               # architecture decision records
└── docker-compose.yml      # local stack (empty for now)
```

## Model

**Dataset:** IEEE-CIS Fraud Detection ([Kaggle competition](https://www.kaggle.com/competitions/ieee-fraud-detection/data)) — 590,540 transactions, 3.5% fraud base rate.

**Algorithm:** LightGBM classifier, `scale_pos_weight` set to the negative/positive ratio to handle class imbalance. Exported to ONNX via `onnxmltools` so the inference service can run without Python.

Two model versions live in [ml/models/](ml/models/):

| Version | Features | ROC-AUC | Threshold | Precision | Recall |
|---------|----------|---------|-----------|-----------|--------|
| v1      | 18 curated | 0.897 | 0.83 | 0.785 | 0.327 |
| v2      | full IEEE-CIS feature set | — | 0.83 | — | — |

Operating threshold is tuned toward precision — the deployed policy needs to justify each flag it produces, so recall is the trade.

### Training

```zsh
cd ml
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# place train_transaction.csv and train_identity.csv into ml/data/
python training/train.py        # curated 18-feature v1
python training/train_full.py   # full-feature v2
python training/evaluate.py     # compare v1 vs v2 on held-out split
```

macOS note: LightGBM's wheel dynamically loads OpenMP. If `import lightgbm` fails with `Library not loaded: @rpath/libomp.dylib`, install it with `brew install libomp`.

## Services (roadmap)

Each service is a standalone FastAPI app. They are wired together only over HTTP / Kafka; no shared Python package.

| Week | Service | Responsibility |
|------|---------|---------------|
| 2 | [services/inference/](services/inference/) | Load `txwatch_v*.onnx`, run scoring, attach SHAP top-k contributions |
| 3 | [services/feature/](services/feature/) | Redis-backed velocity windows (count / amount over rolling windows) |
| 3 | [services/decision/](services/decision/) | Combine model score + rules into `allow` / `review` / `block` |
| 3 | [services/gateway/](services/gateway/) | Public HTTP entrypoint, idempotency key handling, request fan-out |
| 4 | [services/audit/](services/audit/) | Consume decision events from Kafka, persist to PostgreSQL |

## Development

Python 3.13. Each service will carry its own `pyproject.toml` and virtualenv — deliberately no monorepo tooling until we have more than one service to share.

Local dependencies (Redis, Kafka, PostgreSQL) will be brought up via [docker-compose.yml](docker-compose.yml) once the audit and feature services need them.
