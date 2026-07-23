from __future__ import annotations

from orca.hash.common import hash_v2, stable_json_bytes
from orca.hash.verify import verify_graph_hash, verify_instance_hash


class TestCanonicalJSON:
    def test_dict_keys_are_sorted(self):
        data = {"z": 1, "a": 2, "m": 3}
        result = stable_json_bytes(data)
        assert result == b'{"a":2,"m":3,"z":1}'

    def test_nested_dicts_sorted(self):
        data = {"outer": {"z": 1, "a": 2}}
        result = stable_json_bytes(data)
        assert result == b'{"outer":{"a":2,"z":1}}'

    def test_floats_are_preserved(self):
        data = {"price": 100.5}
        result = stable_json_bytes(data)
        assert b"100.5" in result

    def test_nan_is_rejected(self):
        import math

        import pytest

        with pytest.raises(ValueError):
            stable_json_bytes({"val": float("nan")})

    def test_inf_is_rejected(self):
        import pytest

        with pytest.raises(ValueError):
            stable_json_bytes({"val": float("inf")})

    def test_bytes_are_rejected(self):
        import pytest

        with pytest.raises(ValueError):
            stable_json_bytes({"val": b"binary"})


class TestHashing:
    def test_hash_v2_produces_sha256_prefix(self):
        h = hash_v2("test", {"key": "value"})
        assert h.startswith("sha256:")
        assert len(h) == 71  # "sha256:" + 64 hex chars

    def test_same_input_produces_same_hash(self):
        h1 = hash_v2("test", {"a": 1, "b": 2})
        h2 = hash_v2("test", {"b": 2, "a": 1})
        assert h1 == h2

    def test_different_kind_produces_different_hash(self):
        h1 = hash_v2("kind_a", {"x": 1})
        h2 = hash_v2("kind_b", {"x": 1})
        assert h1 != h2

    def test_different_payload_produces_different_hash(self):
        h1 = hash_v2("test", {"x": 1})
        h2 = hash_v2("test", {"x": 2})
        assert h1 != h2

    def test_verify_graph_hash(self, sample_strategy_ir):
        from orca.hash.graph import graph_hash_v2

        h = graph_hash_v2(sample_strategy_ir)
        assert verify_graph_hash(sample_strategy_ir, h) is True
        assert verify_graph_hash(sample_strategy_ir, "sha256:bad") is False

    def test_verify_instance_hash(self, sample_strategy_ir):
        from orca.hash.graph import instance_hash_v2

        h = instance_hash_v2(sample_strategy_ir)
        assert verify_instance_hash(sample_strategy_ir, h) is True
        assert verify_instance_hash(sample_strategy_ir, "sha256:bad") is False
