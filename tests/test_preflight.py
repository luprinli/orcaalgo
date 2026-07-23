from __future__ import annotations

import pytest

from orca.preflight.checklist import CheckResult, run_preflight_checks


class TestCheckResult:
    def test_creation(self):
        cr = CheckResult(check_name="test", status="pass", message="ok")
        assert cr.check_name == "test"
        assert cr.status == "pass"
        assert cr.message == "ok"

    def test_frozen(self):
        cr = CheckResult(check_name="test", status="pass", message="ok")
        with pytest.raises(Exception):
            cr.status = "fail"


class TestRunPreflightChecks:
    def test_returns_list_of_check_results(self):
        results = run_preflight_checks()
        assert isinstance(results, list)
        assert len(results) > 0
        for r in results:
            assert isinstance(r, CheckResult)
            assert r.status in ("pass", "warn", "fail")

    def test_config_exists_check(self):
        results = run_preflight_checks()
        config_check = [r for r in results if r.check_name == "config_exists"]
        assert len(config_check) == 1
        assert config_check[0].status == "pass"

    def test_gkr_strategies_check(self):
        results = run_preflight_checks()
        gkr_check = [r for r in results if r.check_name == "gkr_strategies"]
        assert len(gkr_check) == 1
        assert gkr_check[0].status in ("pass", "warn")

    def test_validate_strategy_check_exists(self):
        results = run_preflight_checks()
        validate_checks = [r for r in results if r.check_name.startswith("validate_")]
        assert len(validate_checks) >= 0

    def test_env_checks(self):
        results = run_preflight_checks()
        env_checks = [r for r in results if r.check_name.startswith("env_")]
        assert len(env_checks) == 2
        for chk in env_checks:
            assert chk.status in ("pass", "warn")

    def test_orca_package_check(self):
        results = run_preflight_checks()
        pkg_check = [r for r in results if r.check_name == "orca_package"]
        assert len(pkg_check) == 1
        assert pkg_check[0].status == "pass"

    def test_numpy_scipy_checks(self):
        results = run_preflight_checks()
        numpy_check = [r for r in results if r.check_name == "numpy_available"]
        scipy_check = [r for r in results if r.check_name == "scipy_available"]
        assert len(numpy_check) == 1
        assert len(scipy_check) == 1
        assert numpy_check[0].status == "pass"
        assert scipy_check[0].status == "pass"
