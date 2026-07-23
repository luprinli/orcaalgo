"""Feature flag system for Python (orca/).

Feature flags isolate new features behind toggles, leaving existing
code paths untouched until the feature is production-ready.

Usage:
    from orca.config.feature_flags import FeatureFlags
    flags = FeatureFlags()
    if flags.is_enabled("new_kill_switch"):
        ...
"""

from pydantic import BaseModel, ConfigDict


class FeatureFlags(BaseModel):
    """Immutable feature flag configuration.

    All flags default to False (off). Enable flags per-environment
    in config.dev.yaml or config.prod.yaml.
    """

    model_config = ConfigDict(frozen=True, extra="forbid")

    new_kill_switch: bool = False
    ibkr_adapter: bool = False
    fix_ingest: bool = False
    llm_analyst: bool = False
    hmm_v2: bool = False

    def is_enabled(self, flag_name: str) -> bool:
        """Check if a feature flag is enabled by name.

        Returns False for unrecognized flag names (safe default).
        """
        return getattr(self, flag_name, False)
