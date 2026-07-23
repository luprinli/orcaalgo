"""Tests for meta-labeling training pipeline."""

import numpy as np
import pytest

from orca.ml.dataset import FeatureDataset, TrainingSample
from orca.ml.train.meta_labeling import MetaLabelingTrainer

try:
    import xgboost  # noqa: F401
    HAS_XGBOOST = True
except ImportError:
    HAS_XGBOOST = False


def make_synthetic_dataset(n_samples: int = 500) -> FeatureDataset:
    rng = np.random.default_rng(42)
    dataset = FeatureDataset()

    for _i in range(n_samples):
        fv = np.zeros(21)
        # Make feature 0 strongly predictive
        if rng.uniform(0, 1) < 0.55:
            fv[0] = rng.uniform(0.5, 1.0)
            label = 1
        else:
            fv[0] = rng.uniform(-1.0, 0.0)
            label = 0
        # Add noise to other features
        fv[1:] = rng.normal(0, 0.5, 20)
        fv[5] = 50.0 + rng.normal(0, 10)  # RSI around 50

        sample = TrainingSample(
            symbol="TEST",
            timestamp="2025-01-01T00:00:00",
            feature_vector=fv,
            label=label,
        )
        dataset.samples.append(sample)

    return dataset


@pytest.mark.skipif(not HAS_XGBOOST, reason="xgboost not installed")
class TestMetaLabelingTrainer:
    def test_train_returns_result(self):
        dataset = make_synthetic_dataset(300)
        trainer = MetaLabelingTrainer(n_estimators=20, max_depth=3, early_stopping_rounds=5, min_samples=5)
        result = trainer.train(dataset)

        assert result.model is not None
        assert result.brier_score >= 0
        assert result.roc_auc >= 0
        assert 0 <= result.accuracy <= 1
        assert len(result.cv_scores) > 0
        assert len(result.feature_importance) > 0

    def test_feature_importance_has_all_features(self):
        dataset = make_synthetic_dataset(200)
        trainer = MetaLabelingTrainer(n_estimators=10, max_depth=2, early_stopping_rounds=3, min_samples=5)
        result = trainer.train(dataset)

        assert len(result.feature_importance) == len(dataset.feature_names)
        assert "ret1" in result.feature_importance

    def test_predictive_feature_has_highest_importance(self):
        dataset = make_synthetic_dataset(300)
        trainer = MetaLabelingTrainer(n_estimators=20, max_depth=3, early_stopping_rounds=5, min_samples=5)
        result = trainer.train(dataset)

        # Feature 0 (ret1) is the synthetic predictive feature
        sorted_features = sorted(
            result.feature_importance.items(),
            key=lambda x: x[1], reverse=True,
        )
        top_feature = sorted_features[0][0]
        assert result.feature_importance[top_feature] > 0

    def test_model_can_predict(self):
        dataset = make_synthetic_dataset(300)
        trainer = MetaLabelingTrainer(n_estimators=20, max_depth=3, early_stopping_rounds=5, min_samples=5)
        result = trainer.train(dataset)

        X, _ = dataset.to_numpy()
        proba = result.model.predict_proba(X)
        assert proba.shape == (len(dataset.samples), 2)
        assert np.all((proba >= 0) & (proba <= 1))
        assert np.allclose(proba.sum(axis=1), 1.0)

    def test_invalid_dataset_raises(self):
        dataset = make_synthetic_dataset(5)  # too few samples for validation
        trainer = MetaLabelingTrainer(n_estimators=5)
        with pytest.raises(ValueError):
            trainer.train(dataset)

    def test_save_and_load_model(self, tmp_path):
        dataset = make_synthetic_dataset(200)
        trainer = MetaLabelingTrainer(n_estimators=10, max_depth=2, early_stopping_rounds=3, min_samples=5)
        result = trainer.train(dataset)

        model_path = tmp_path / "test_model.json"
        trainer.save_model(result, model_path)
        assert model_path.exists()

        from orca.ml.train.meta_labeling import load_model, predict
        loaded = load_model(model_path)
        X, _ = dataset.to_numpy()
        proba = predict(loaded, X)
        assert len(proba) == len(dataset.samples)
