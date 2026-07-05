import time
import numpy as np
import shap
from typing import Optional

class SHAPStrategyBase:
    """Base class for all SHAP strategies."""
    
    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        """
        Returns:
            - shap_values: dict of feature_index -> shap_value
            - computation_ms: time taken in milliseconds
        """
        raise NotImplementedError


class ExactTreeSHAP(SHAPStrategyBase):
    """Exact TreeSHAP with full background dataset (ground truth)."""
    
    def __init__(self, model, background_data: np.ndarray):
        print(f"Initialising ExactTreeSHAP with {len(background_data)} background samples...")
        self.explainer = shap.TreeExplainer(
            model,
            data=shap.maskers.Independent(background_data, max_samples=1000),
            feature_perturbation="interventional"
        )
        print("ExactTreeSHAP ready.")

    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        start = time.perf_counter()
        shap_values = self.explainer.shap_values(features)
        ms = (time.perf_counter() - start) * 1000

        # shap_values is list [legit_shap, fraud_shap] — take fraud class
        fraud_shap = shap_values[1][0] if isinstance(shap_values, list) else shap_values[0]
        return {str(i): float(v) for i, v in enumerate(fraud_shap)}, round(ms, 3)


class ReducedBackgroundTreeSHAP(SHAPStrategyBase):
    """TreeSHAP with sub-sampled background dataset."""

    def __init__(self, model, background_data: np.ndarray, n_background: int):
        print(f"Initialising ReducedBackgroundTreeSHAP with {n_background} background samples...")
        indices = np.random.choice(len(background_data), size=n_background, replace=False)
        reduced_background = background_data[indices]
        self.explainer = shap.TreeExplainer(
            model,
            data=shap.maskers.Independent(reduced_background, max_samples=n_background),
            feature_perturbation="interventional"
        )
        print(f"ReducedBackgroundTreeSHAP ({n_background}) ready.")

    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        start = time.perf_counter()
        shap_values = self.explainer.shap_values(features)
        ms = (time.perf_counter() - start) * 1000
        fraud_shap = shap_values[1][0] if isinstance(shap_values, list) else shap_values[0]
        return {str(i): float(v) for i, v in enumerate(fraud_shap)}, round(ms, 3)


class FeatureSampledSHAP(SHAPStrategyBase):
    """TreeSHAP computed only for top-k features by global importance."""

    def __init__(self, model, background_data: np.ndarray, top_k: int = 20):
        print(f"Initialising FeatureSampledSHAP with top {top_k} features...")
        self.explainer = shap.TreeExplainer(
            model,
            data=shap.maskers.Independent(background_data, max_samples=1000),
            feature_perturbation="interventional"
        )
        self.top_k = top_k
        self.n_features = background_data.shape[1]

        # Get global feature importance to identify top-k
        sample = background_data[:100]
        sample_shap = self.explainer.shap_values(sample)
        sample_fraud = sample_shap[1] if isinstance(sample_shap, list) else sample_shap
        mean_abs = np.abs(sample_fraud).mean(axis=0)
        self.top_indices = set(np.argsort(mean_abs)[-top_k:].tolist())
        print(f"FeatureSampledSHAP ready. Top {top_k} features identified.")

    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        start = time.perf_counter()
        shap_values = self.explainer.shap_values(features)
        ms = (time.perf_counter() - start) * 1000
        fraud_shap = shap_values[1][0] if isinstance(shap_values, list) else shap_values[0]

        # Zero out non-top-k features
        result = {
            str(i): float(v) if i in self.top_indices else 0.0
            for i, v in enumerate(fraud_shap)
        }
        return result, round(ms, 3)


class KernelSHAP(SHAPStrategyBase):
    """Model-agnostic KernelSHAP — slowest but framework-independent."""

    def __init__(self, predict_fn, background_data: np.ndarray, n_background: int = 50):
        print(f"Initialising KernelSHAP with {n_background} background samples...")
        indices = np.random.choice(len(background_data), size=n_background, replace=False)
        reduced_background = background_data[indices]
        self.explainer = shap.KernelExplainer(predict_fn, reduced_background)
        print("KernelSHAP ready.")

    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        start = time.perf_counter()
        shap_values = self.explainer.shap_values(features, nsamples=100, silent=True)
        ms = (time.perf_counter() - start) * 1000
        fraud_shap = shap_values[1][0] if isinstance(shap_values, list) else shap_values[0]
        return {str(i): float(v) for i, v in enumerate(fraud_shap)}, round(ms, 3)


class AsyncPostHocSHAP(SHAPStrategyBase):
    """
    Returns zero SHAP values immediately.
    Actual computation is dispatched as a FastAPI background task.
    """

    def __init__(self, model, background_data: np.ndarray):
        print("Initialising AsyncPostHocSHAP...")
        self.explainer = shap.TreeExplainer(
            model,
            data=shap.maskers.Independent(background_data[:100], max_samples=100),
            feature_perturbation="interventional"
        )
        print("AsyncPostHocSHAP ready.")

    def compute(self, features: np.ndarray) -> tuple[dict, float]:
        # Returns immediately — no SHAP on hot path
        n_features = features.shape[1]
        return {str(i): 0.0 for i in range(n_features)}, 0.0

    def compute_background(self, features: np.ndarray) -> dict:
        """Called from FastAPI BackgroundTasks after response is sent."""
        shap_values = self.explainer.shap_values(features)
        fraud_shap = shap_values[1][0] if isinstance(shap_values, list) else shap_values[0]
        return {str(i): float(v) for i, v in enumerate(fraud_shap)}


def load_strategy(strategy_name: str, model, background_data: np.ndarray, predict_fn=None):
    """Factory function — loads the configured SHAP strategy."""
    strategies = {
        "exact_tree": lambda: ExactTreeSHAP(model, background_data),
        "reduced_tree_50": lambda: ReducedBackgroundTreeSHAP(model, background_data, 50),
        "reduced_tree_100": lambda: ReducedBackgroundTreeSHAP(model, background_data, 100),
        "reduced_tree_200": lambda: ReducedBackgroundTreeSHAP(model, background_data, 200),
        "feature_sampled": lambda: FeatureSampledSHAP(model, background_data),
        "kernel": lambda: KernelSHAP(predict_fn, background_data),
        "async_posthoc": lambda: AsyncPostHocSHAP(model, background_data),
    }

    if strategy_name not in strategies:
        raise ValueError(f"Unknown SHAP strategy: {strategy_name}. Choose from {list(strategies.keys())}")

    return strategies[strategy_name]()
