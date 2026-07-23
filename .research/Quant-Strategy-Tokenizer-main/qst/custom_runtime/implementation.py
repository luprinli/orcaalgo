"""Implementation and runtime-environment references for WP9 custom tokens."""

from __future__ import annotations

import base64
import hashlib
import importlib.metadata
import platform
import sys
from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from qst.canonical_json import stable_json_bytes
from qst.hash import implementation_ref_hash_v2, runtime_environment_hash_v2

ImplementationRefKind = Literal[
    "spec_only",
    "source_tree",
    "wheel",
    "sdist",
    "installed_distribution",
]
ReproducibilityLevel = Literal[
    "environment_recorded",
    "environment_replayable",
    "bit_exact_verified",
]


class ImplementationRef(BaseModel):
    """Executable implementation identity material.

    The model is metadata only. Validating it must not import or execute the
    referenced Python module.
    """

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-implementation-ref/0.4"] = "qst-implementation-ref/0.4"
    kind: ImplementationRefKind
    python_entrypoint: str | None = None
    path: str | None = None
    distribution: str | None = None
    expected_hash: str | None = Field(default=None, pattern=r"^sha256:[0-9a-f]{64}$")
    editable: bool = False

    @model_validator(mode="after")
    def _validate_shape(self) -> ImplementationRef:
        if self.kind == "spec_only" and self.python_entrypoint is not None:
            raise ValueError("spec_only implementation_ref cannot declare python_entrypoint")
        if self.kind in {"source_tree", "wheel", "sdist"} and self.path is None:
            raise ValueError(f"{self.kind} implementation_ref requires path")
        if self.kind == "installed_distribution" and self.distribution is None:
            raise ValueError("installed_distribution implementation_ref requires distribution")
        if self.editable and self.kind != "source_tree":
            raise ValueError("editable installs must be converted to source_tree references")
        if self.python_entrypoint is not None and ":" not in self.python_entrypoint:
            raise ValueError("python_entrypoint must be formatted as module:function")
        return self

    def material(self) -> dict[str, Any]:
        """Canonical hash material."""

        return self.model_dump(mode="json", exclude_none=True)


class RuntimeEnvironmentRef(BaseModel):
    """Runtime environment identity material for custom tokens."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_version: Literal["qst-runtime-environment-ref/0.4"] = (
        "qst-runtime-environment-ref/0.4"
    )
    python_version: str
    platform_marker: str
    distribution_hashes: dict[str, str] = Field(default_factory=dict)
    dependency_token_pack_hashes: dict[str, str] = Field(default_factory=dict)
    package_manager: dict[str, Any] = Field(default_factory=dict)
    reproducibility_level: ReproducibilityLevel = "environment_recorded"

    @model_validator(mode="after")
    def _validate_json(self) -> RuntimeEnvironmentRef:
        stable_json_bytes(self.model_dump(mode="json"))
        return self


def implementation_ref_hash_for_ref(ref: ImplementationRef | dict[str, Any] | None) -> str:
    """Hash an implementation reference model or payload."""

    if ref is None:
        return implementation_ref_hash_v2(None)
    parsed = ref if isinstance(ref, ImplementationRef) else ImplementationRef.model_validate(ref)
    return implementation_ref_hash_v2(parsed.material())


def runtime_environment_ref_hash_for_ref(
    ref: RuntimeEnvironmentRef | dict[str, Any] | None,
) -> str:
    """Hash a runtime-environment reference model or payload."""

    if ref is None:
        return runtime_environment_hash_v2(None)
    parsed = ref if isinstance(ref, RuntimeEnvironmentRef) else RuntimeEnvironmentRef.model_validate(ref)
    return runtime_environment_hash_v2(parsed.model_dump(mode="json"))


def runtime_environment_ref_current(
    *,
    dependency_token_pack_hashes: dict[str, str] | None = None,
    distribution_hashes: dict[str, str] | None = None,
) -> RuntimeEnvironmentRef:
    """Build a minimal current environment reference without importing custom code."""

    return RuntimeEnvironmentRef(
        python_version=f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
        platform_marker=f"{platform.system().lower()}-{platform.machine().lower()}",
        distribution_hashes=distribution_hashes or {},
        dependency_token_pack_hashes=dependency_token_pack_hashes or {},
        package_manager={"kind": "python"},
        reproducibility_level=(
            "environment_replayable" if distribution_hashes else "environment_recorded"
        ),
    )


def resolve_implementation_hash(ref: ImplementationRef, *, base_path: Path) -> tuple[str, str | None]:
    """Compute or return the concrete implementation hash.

    Returns ``(hash, diagnostic_code)``. The function is intentionally file and
    metadata based. It never imports a custom package.
    """

    if ref.kind == "spec_only":
        return implementation_ref_hash_for_ref(ref), None
    if ref.editable and ref.kind != "source_tree":
        return implementation_ref_hash_for_ref(ref), "QST_V2_EDITABLE_INSTALL_REQUIRES_SOURCE_TREE"
    if ref.kind == "source_tree":
        assert ref.path is not None
        computed = source_tree_hash(base_path / ref.path)
        return computed, None
    if ref.kind in {"wheel", "sdist"}:
        assert ref.path is not None
        path = base_path / ref.path
        if not path.exists():
            return implementation_ref_hash_for_ref(ref), "QST_V2_IMPLEMENTATION_REF_MISSING"
        return "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest(), None
    if ref.kind == "installed_distribution":
        assert ref.distribution is not None
        try:
            distribution = importlib.metadata.distribution(ref.distribution)
        except importlib.metadata.PackageNotFoundError:
            return implementation_ref_hash_for_ref(ref), "QST_V2_DISTRIBUTION_NOT_INSTALLED"
        files = distribution.files or ()
        if not files:
            return implementation_ref_hash_for_ref(ref), "QST_V2_DISTRIBUTION_RECORD_INCOMPLETE"
        incomplete_record = False
        record_hash_mismatch = False
        file_entries: list[dict[str, Any]] = []
        for file in sorted(files, key=str):
            entry: dict[str, Any] = {"path": str(file)}
            size = getattr(file, "size", None)
            if size is not None:
                entry["size"] = size
            try:
                installed_path = Path(str(distribution.locate_file(file)))
                file_bytes = installed_path.read_bytes()
            except OSError:
                return (
                    implementation_ref_hash_v2(
                        {
                            "kind": ref.kind,
                            "distribution": ref.distribution,
                            "version": distribution.version,
                            "record_unreadable": str(file),
                        }
                    ),
                    "QST_V2_DISTRIBUTION_RECORD_INCOMPLETE",
                )
            entry["sha256"] = hashlib.sha256(file_bytes).hexdigest()
            record_hash = getattr(file, "hash", None)
            if record_hash is not None:
                mode = getattr(record_hash, "mode", None)
                value = getattr(record_hash, "value", None)
                mode_text = str(mode) if mode is not None else "unknown"
                value_text = str(value) if value is not None else str(record_hash)
                entry["record_hash"] = {
                    "mode": mode_text,
                    "value": value_text,
                }
                if mode_text != "sha256":
                    incomplete_record = True
                elif _record_sha256_value(file_bytes) != value_text.rstrip("="):
                    record_hash_mismatch = True
            else:
                incomplete_record = True
            file_entries.append(entry)
        material = {
            "kind": ref.kind,
            "distribution": ref.distribution,
            "version": distribution.version,
            "files": file_entries,
        }
        diagnostic = None
        if record_hash_mismatch:
            diagnostic = "QST_V2_DISTRIBUTION_RECORD_HASH_MISMATCH"
        elif incomplete_record:
            diagnostic = "QST_V2_DISTRIBUTION_RECORD_INCOMPLETE"
        return implementation_ref_hash_v2(material), diagnostic
    raise AssertionError(f"Unhandled implementation kind: {ref.kind}")


def source_tree_hash(path: Path) -> str:
    """Hash a source tree deterministically without importing it."""

    if not path.exists() or not path.is_dir():
        raise FileNotFoundError(f"source_tree path does not exist: {path}")
    entries: list[dict[str, str]] = []
    for file_path in sorted(item for item in path.rglob("*") if item.is_file()):
        if "__pycache__" in file_path.parts or file_path.suffix in {".pyc", ".pyo"}:
            continue
        relative = file_path.relative_to(path).as_posix()
        digest = hashlib.sha256(file_path.read_bytes()).hexdigest()
        entries.append({"path": relative, "sha256": digest})
    return implementation_ref_hash_v2({"kind": "source_tree", "files": entries})


def _record_sha256_value(file_bytes: bytes) -> str:
    digest = hashlib.sha256(file_bytes).digest()
    return base64.urlsafe_b64encode(digest).decode("ascii").rstrip("=")
