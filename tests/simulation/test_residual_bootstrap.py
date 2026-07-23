"""Unit tests for residual bootstrap module."""

import numpy as np

from orca.simulation.residual_bootstrap import ResidualBootstrap


def test_bootstrap_returns_correct_shape() -> None:
    real_rets = np.random.default_rng(42).normal(0.0005, 0.01, 500)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None)
    boot = rb.bootstrap(n_paths=3, seed=42)
    assert boot.shape == (3, 500), f"Expected (3, 500), got {boot.shape}"


def test_bootstrap_single_path() -> None:
    real_rets = np.random.default_rng(42).normal(0.0005, 0.01, 500)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None)
    boot = rb.bootstrap(n_paths=1, seed=42)
    assert boot.shape == (1, 500)


def test_bootstrap_mean_preserved() -> None:
    rng = np.random.default_rng(42)
    real_mean = 0.0005
    real_rets = rng.normal(real_mean, 0.01, 500)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None)
    boot = rb.bootstrap(n_paths=1, seed=42)
    boot_mean = float(np.mean(boot))
    # Mean should be near the original mean (drift preservation)
    assert abs(boot_mean - real_mean) < 0.005, f"Mean drift: {boot_mean:.6f} vs {real_mean:.6f}"


def test_bootstrap_block_size() -> None:
    real_rets = np.random.default_rng(42).normal(0.0, 0.01, 200)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None, block_size=50)
    boot = rb.bootstrap(n_paths=1, seed=42)
    assert boot.shape == (1, 200)


def test_bootstrap_multiple_calls_different() -> None:
    real_rets = np.random.default_rng(42).normal(0.0, 0.01, 500)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None)
    boot1 = rb.bootstrap(n_paths=1, seed=42)
    boot2 = rb.bootstrap(n_paths=1, seed=99)
    assert not np.allclose(boot1, boot2), "Different seeds must produce different results"


def test_bootstrap_deterministic_with_seed() -> None:
    real_rets = np.random.default_rng(42).normal(0.0, 0.01, 500)
    rb = ResidualBootstrap(real_rets.reshape(-1, 1), None)
    boot1 = rb.bootstrap(n_paths=1, seed=42)
    boot2 = rb.bootstrap(n_paths=1, seed=42)
    assert np.allclose(boot1, boot2), "Same seed must produce identical results"
