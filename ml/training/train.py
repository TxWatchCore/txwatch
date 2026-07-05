from pathlib import Path

import pandas as pd
import numpy as np
import lightgbm as lgb
import pickle

from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report, precision_recall_curve, roc_auc_score
from sklearn.metrics import precision_score, recall_score
from sklearn.preprocessing import LabelEncoder

from onnxmltools import convert_lightgbm
from onnxmltools.convert.common.data_types import FloatTensorType
from matplotlib import pyplot as plt

DATA_DIR = Path(__file__).resolve().parent.parent / "data"

print("Loading data...")
tx = pd.read_csv(DATA_DIR / "train_transaction.csv")
id = pd.read_csv(DATA_DIR / "train_identity.csv")

df = tx.merge(id, on='TransactionID', how='left')
print(f"Dataset shape: {df.shape}")
print(f"Fraud rate: {df['isFraud'].mean():.4%}")


FEATURES = [
    "TransactionAmt",
    "ProductCD",
    "card4",       # card network: visa, mastercard etc
    "card6",       # debit/credit
    "P_emaildomain",
    "R_emaildomain",
    "C1", "C2", "C3", "C4",   # counting features (obfuscated by Kaggle)
    "C5", "C6", "C7", "C8",
    "V1", "V2", "V3",          # Vesta engineered features
    "TransactionDT",           # time delta — proxy for time of day
]

TARGET = "isFraud"
df = df[FEATURES + [TARGET]].copy()


print("Preprocessing...")
cat_col = ["ProductCD", "card4", "card6", "P_emaildomain", "R_emaildomain"]
encoders = {}

for col in cat_col:
    le = LabelEncoder()
    df[col] = df[col].astype(str).fillna("unknown")
    df[col] = le.fit_transform(df[col])
    encoders[col] = le

df = df.fillna(-999)


X = df[FEATURES]
y = df[TARGET]

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42, stratify=y)

print(f"Train size: {len(X_train)}, Test size: {len(X_test)}")


print("Training model...")
neg = (y_train == 0).sum()
pos = (y_train == 1).sum()
scale = neg / pos

model = lgb.LGBMClassifier(
    n_estimators=500,
    learning_rate=0.05,
    num_leaves=31,
    scale_pos_weight=scale,
    random_state=42,
    n_jobs=-1
)

model.fit(X_train, y_train, eval_set=[(X_test, y_test)])

print("\nEvaluating...")
y_prob = model.predict_proba(X_test)[:, 1]

# TODO: make threshold configurable via environment variable
# Current value chosen for production-conservative behaviour
# Precision: 0.7850, Recall: 0.3269 at this threshold
OPERATING_THRESHOLD = 0.83
y_pred = (y_prob >= OPERATING_THRESHOLD).astype(int)

print(f"\nClassification report (threshold={OPERATING_THRESHOLD}):")
print(classification_report(y_test, y_pred, target_names=["legit", "fraud"]))
print(f"ROC-AUC: {roc_auc_score(y_test, y_prob):.4f}")


# Precision-recall curve — pick your operating threshold
precision, recall, thresholds = precision_recall_curve(y_test, y_prob)

diff = np.abs(precision[:-1] - recall[:-1])
crossover_idx = diff.argmin()
crossover_threshold = thresholds[crossover_idx]
crossover_precision = precision[crossover_idx]
crossover_recall = recall[crossover_idx]

print(f"\nCrossover point:")
print(f"  Threshold: {crossover_threshold:.4f}")
print(f"  Precision: {crossover_precision:.4f}")
print(f"  Recall:    {crossover_recall:.4f}")


y_pred_operating = (y_prob >= OPERATING_THRESHOLD).astype(int)
exact_precision = precision_score(y_test, y_pred_operating)
exact_recall = recall_score(y_test, y_pred_operating)

print(f"\nExact metrics at threshold {OPERATING_THRESHOLD}:")
print(f"  Precision: {exact_precision:.4f}")
print(f"  Recall:    {exact_recall:.4f}")

plt.figure(figsize=(8, 5))
plt.plot(thresholds, precision[:-1], label="Precision")
plt.plot(thresholds, recall[:-1], label="Recall")
plt.xlabel("Threshold")
plt.ylabel("Score")
plt.title("TxWatch — Precision / Recall vs Threshold")
plt.legend()
plt.grid(True)
plt.tight_layout()
plt.savefig("../models/precision_recall_curve.png")
print("Saved precision/recall curve to ml/models/precision_recall_curve.png")


print("\nExporting to ONNX...")

initial_type = [("float_input", FloatTensorType([None, len(FEATURES)]))]
onnx_model = convert_lightgbm(model, initial_types=initial_type, target_opset=12)

with open("../models/txwatch_v1.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())

print("Model saved to ml/models/txwatch_v1.onnx")

with open("../models/encoders.pkl", "wb") as f:
    pickle.dump(encoders, f)

print("Encoders saved to ml/models/encoders.pkl")
print("\nDone.")
