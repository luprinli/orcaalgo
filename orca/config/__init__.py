"""Feature flag system for Python (orca/).

Feature flags isolate new features behind toggles, leaving existing
code paths untouched until the feature is production-ready.

Usage:
    from orca.config.feature_flags import FeatureFlags
    flags = FeatureFlags()
    if flags.is_enabled("new_kill_switch"):
        ...
"""

from orca.config.feature_flags import FeatureFlags

__all__ = ["FeatureFlags"]
