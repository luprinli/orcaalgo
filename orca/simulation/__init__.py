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
    # regime
    "DEFAULT_TRANSITION_MATRIX",
    "REGIME_CALM",
    "REGIME_CRISIS",
    "REGIME_FACTORS",
    "REGIME_HIGH_VOL",
    "REGIME_NAMES",
    "REGIME_TRENDING",
    "BreakoutInjector",
    # multi-factor generation (Phase 2)
    "FactorGenerator",
    "MeanReversionInjector",
    "RegimeBatchState",
    "RegimeParams",
    "RegimeSequenceGenerator",
    # residual bootstrap (Phase 3)
    "ResidualBootstrap",
    # signal injection (Phase 1)
    "TrendInjector",
    "bootstrap_generate",
    # calibration
    "calibrate_all",
    "calibrate_per_regime",
    "calibrate_symbol",
    # tick disaggregation
    "disaggregate_1m_to_ticks",
    "disaggregate_and_save",
    # generation
    "generate_1m_candles",
    "generate_1m_candles_from_factors",
    "generate_and_save",
    "generate_regime_aware",
    "generate_regime_ticks",
    "load_real_candles",
    "load_regime_params",
    "regime_params_for_state",
    "save_regime_params",
    # validation
    "validate_generation",
    "validate_strategy_coverage",
]
