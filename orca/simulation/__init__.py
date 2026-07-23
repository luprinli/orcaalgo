"""Synthetic data simulation pipeline.

Provides calibration, 1-minute generation, tick disaggregation,
regime-aware generation, signal injection, multi-factor generation,
residual bootstrap, and statistical validation for realistic
synthetic market data.

Regime-aware modules (new):
  - regime.py: regime sequence generator, parameter mapping, batch progress
  - calibrate_regime.py: per-regime parameter calibration
  - regime_generator.py: regime-conditioned synthetic data generation

Signal coverage modules:
  - signal_injector.py: inject structured alpha into existing Heston paths
  - factor_generator.py: multi-factor regime generation (trend, MR, vol)
  - residual_bootstrap.py: empirical residual bootstrap from real data
"""

from orca.simulation.calibrate import calibrate_all, calibrate_symbol, load_real_candles
from orca.simulation.calibrate_regime import (
    calibrate_per_regime,
    load_regime_params,
    save_regime_params,
)
from orca.simulation.factor_generator import (
    REGIME_FACTORS,
    FactorGenerator,
    generate_1m_candles_from_factors,
)
from orca.simulation.generate_1m import generate_1m_candles, generate_and_save
from orca.simulation.regime import (
    DEFAULT_TRANSITION_MATRIX,
    REGIME_CALM,
    REGIME_CRISIS,
    REGIME_HIGH_VOL,
    REGIME_NAMES,
    REGIME_TRENDING,
    RegimeBatchState,
    RegimeParams,
    RegimeSequenceGenerator,
    regime_params_for_state,
)
from orca.simulation.regime_generator import (
    generate_regime_aware,
    generate_regime_ticks,
)
from orca.simulation.residual_bootstrap import ResidualBootstrap, bootstrap_generate
from orca.simulation.signal_injector import BreakoutInjector, MeanReversionInjector, TrendInjector
from orca.simulation.tick_disaggregator import disaggregate_1m_to_ticks, disaggregate_and_save
from orca.simulation.validate import validate_generation, validate_strategy_coverage

__all__ = [
    # calibration
    "calibrate_all",
    "calibrate_symbol",
    "calibrate_per_regime",
    "load_regime_params",
    "load_real_candles",
    "save_regime_params",
    # generation
    "generate_1m_candles",
    "generate_and_save",
    "generate_regime_aware",
    "generate_regime_ticks",
    # tick disaggregation
    "disaggregate_1m_to_ticks",
    "disaggregate_and_save",
    # regime
    "DEFAULT_TRANSITION_MATRIX",
    "REGIME_CALM",
    "REGIME_CRISIS",
    "REGIME_HIGH_VOL",
    "REGIME_NAMES",
    "REGIME_TRENDING",
    "RegimeBatchState",
    "RegimeParams",
    "RegimeSequenceGenerator",
    "regime_params_for_state",
    # validation
    "validate_generation",
    "validate_strategy_coverage",
    # signal injection (Phase 1)
    "TrendInjector",
    "MeanReversionInjector",
    "BreakoutInjector",
    # multi-factor generation (Phase 2)
    "FactorGenerator",
    "generate_1m_candles_from_factors",
    "REGIME_FACTORS",
    # residual bootstrap (Phase 3)
    "ResidualBootstrap",
    "bootstrap_generate",
]
