import onnxruntime as rt
import numpy as np
import pandas as pd
from pathlib import Path
from sklearn.metrics import precision_score, recall_score, f1_score, roc_auc_score
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import LabelEncoder

DATA_DIR = Path(__file__).resolve().parent.parent / "data"

print("Loading data for v1 evaluation...")
tx = pd.read_csv(DATA_DIR / "train_transaction.csv")
id_df = pd.read_csv(DATA_DIR / "train_identity.csv")
df = tx.merge(id_df, on="TransactionID", how="left")

FEATURES_V1 = [
    "TransactionAmt", "ProductCD", "card4", "card6",
    "P_emaildomain", "R_emaildomain",
    "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8",
    "V1", "V2", "V3", "TransactionDT",
]

TARGET = "isFraud"
df_v1 = df[FEATURES_V1 + [TARGET]].copy()

cat_cols = ["ProductCD", "card4", "card6", "P_emaildomain", "R_emaildomain"]
for col in cat_cols:
    le = LabelEncoder()
    df_v1[col] = df_v1[col].astype(str).fillna("unknown")
    df_v1[col] = le.fit_transform(df_v1[col])

df_v1 = df_v1.fillna(-999)

X = df_v1[FEATURES_V1].astype(np.float32)
y = df_v1[TARGET]

_, X_test, _, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

THRESHOLD = 0.83

for version in ["v1", "v2"]:
    model_path = f"../models/txwatch_{version}.onnx"
    try:
        sess = rt.InferenceSession(model_path)
        input_name = sess.get_inputs()[0].name

        if version == "v1":
            X_eval = X_test
        else:
            # Rebuild full feature set for v2
            df_v2 = df.copy()
            drop_cols = ["isFraud", "TransactionID"]
            FEATURES_V2 = [col for col in df_v2.columns if col not in drop_cols]
            df_v2 = df_v2[FEATURES_V2 + [TARGET]].copy()

            for col in df_v2.columns:
                if df_v2[col].dtype == 'object' or str(df_v2[col].dtype) == 'str':
                    le = LabelEncoder()
                    df_v2[col] = df_v2[col].astype(str).fillna("unknown")
                    df_v2[col] = le.fit_transform(df_v2[col])

            df_v2 = df_v2.fillna(-999)

            X2 = df_v2[FEATURES_V2].astype(np.float32)
            y2 = df_v2[TARGET]

            _, X_eval, _, y_eval = train_test_split(
                X2, y2, test_size=0.2, random_state=42, stratify=y2
            )

        preds = sess.run(None, {input_name: X_eval.values})
        y_prob = np.array([p[1] for p in preds[1]])
        y_pred = (y_prob >= THRESHOLD).astype(int)

        print(f"\n--- {version} at threshold {THRESHOLD} ---")
        print(f"ROC-AUC:   {roc_auc_score(y_test, y_prob):.4f}")
        print(f"Precision: {precision_score(y_test, y_pred):.4f}")
        print(f"Recall:    {recall_score(y_test, y_pred):.4f}")
        print(f"F1:        {f1_score(y_test, y_pred):.4f}")
    except Exception as e:
        print(f"Could not evaluate {version}: {e}")
