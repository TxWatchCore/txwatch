import requests
import pandas as pd
import numpy as np
from sklearn.preprocessing import LabelEncoder
from pathlib import Path
import json

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

# Pick first row as test
row = df[FEATURES].iloc[0].astype(float).tolist()

payload = {
    "transaction_id": "txn_test_001",
    "features": row
}

response = requests.post("http://localhost:8001/infer", json=payload)
print(json.dumps(response.json(), indent=2))
