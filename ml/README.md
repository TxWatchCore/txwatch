# TxWatch ML

This directory contains all model training and evaluation code.
The Go services consume only the output artifact: `models/txwatch_v1.onnx`.

## Dataset
IEEE-CIS Fraud Detection (Kaggle)
Download from: https://www.kaggle.com/competitions/ieee-fraud-detection/data
Place files in `data/` — this directory is gitignored.

## Setup
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

## Training
python training/train.py

## Evaluation
Results documented in notebooks/model_selection.ipynb

Dataset:           IEEE-CIS Fraud Detection (590,540 transactions, 3.5% fraud rate)
ROC-AUC:           0.897
Operating threshold: 0.90
Precision at 0.90: 0.785
Recall at 0.90:    0.327
