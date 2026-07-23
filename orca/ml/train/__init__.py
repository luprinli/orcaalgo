"""Model training pipelines for the Orca ML subsystem.

Tier 1: Meta-labeling — XGBoost binary classifier (false signal filter)
Tier 2: Regime enhancement — XGBoost multi-class (6 regime states)
Tier 3: Exit optimization — LightGBM regressor (dynamic stop multiplier)
"""

__version__ = "0.1.0"
