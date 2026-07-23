"""Stable error kinds for QST."""

from __future__ import annotations

from enum import StrEnum


class ErrorKind(StrEnum):
    """Public error kind identifiers bound by semver."""

    token_not_found = "token_not_found"
    recipe_not_found = "recipe_not_found"
    token_duplicate_registration = "token_duplicate_registration"
    recipe_duplicate_registration = "recipe_duplicate_registration"

    type_mismatch = "type_mismatch"
    missing_input = "missing_input"
    missing_external = "missing_external"
    missing_risk_path = "missing_risk_path"
    missing_unknown_handling = "missing_unknown_handling"
    invalid_params = "invalid_params"
    profile_violation = "profile_violation"
    purity_violation = "purity_violation"
    future_data_violation = "future_data_violation"
    unsafe_temporal_window = "unsafe_temporal_window"
    future_data_warning = "future_data_warning"
    duplicate_node_id = "duplicate_node_id"
    cycle_detected = "cycle_detected"
    max_depth_exceeded = "max_depth_exceeded"
    unsupported_canonical_version = "unsupported_canonical_version"

    executor_exception = "executor_exception"
    executor_error = "executor_error"
    division_by_zero = "division_by_zero"
    zero_range = "zero_range"

    yaml_parse_error = "yaml_parse_error"
    schema_violation = "schema_violation"
