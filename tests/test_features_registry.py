"""Tests for the self-documenting feature registry (R9)."""

from __future__ import annotations

from orca.features import FEATURE_REGISTRY, get_feature_metadata, list_features


def test_registry_non_empty():
    assert len(FEATURE_REGISTRY) >= 5


def test_get_feature_metadata_case_insensitive():
    meta = get_feature_metadata("RSI")
    assert meta is not None
    assert meta["category"] == "momentum"


def test_get_feature_metadata_unknown_is_none():
    assert get_feature_metadata("does_not_exist") is None


def test_list_features_filter_by_category():
    vol = list_features(category="volatility")
    assert all(f["category"] == "volatility" for f in vol)
    assert len(vol) >= 2


def test_list_features_all_includes_key():
    all_features = list_features()
    keys = {f["key"] for f in all_features}
    assert "rsi" in keys
    assert "atr" in keys
