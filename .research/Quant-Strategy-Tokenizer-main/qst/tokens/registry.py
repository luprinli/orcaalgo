"""Token System v2 registry and TokenPack dependency validation."""

from __future__ import annotations

from collections import defaultdict
from collections.abc import Iterable
from dataclasses import dataclass

from pydantic import BaseModel, ConfigDict, Field

from qst.hash.token_pack_hash import token_pack_hash_for_pack_v2
from qst.hash.token_spec_hash import token_spec_hash_for_spec_v2
from qst.tokens.pack import TokenPackDependency, TokenPackManifestV2
from qst.tokens.spec import TokenSpecV2
from qst.validation import Diagnostic, Severity, ValidationResult


class RegistryTokenRecord(BaseModel):
    """Resolved registry token record."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    spec: TokenSpecV2
    pack_id: str
    pack_origin_tier: str
    token_spec_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")
    token_pack_hash: str = Field(pattern=r"^sha256:[0-9a-f]{64}$")


@dataclass(frozen=True)
class TokenPackDependencyResolution:
    """Result of deterministic TokenPack dependency resolution."""

    ordered_packs: tuple[TokenPackManifestV2, ...]
    result: ValidationResult


class TokenRegistryV2(BaseModel):
    """Resolved v2 token registry built from TokenPack manifests."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    records: tuple[RegistryTokenRecord, ...]
    resolution_log: tuple[str, ...]
    result: ValidationResult

    @classmethod
    def from_packs(cls, packs: Iterable[TokenPackManifestV2]) -> TokenRegistryV2:
        """Build a registry from token packs without reading removed registry internals."""

        sorted_packs = _sort_packs(tuple(packs))
        diagnostics = list(validate_token_pack_dependencies(sorted_packs).result.diagnostics)
        records: dict[tuple[str, str, int, int], RegistryTokenRecord] = {}
        resolution_log: list[str] = []

        for pack in sorted_packs:
            pack_hash = token_pack_hash_for_pack_v2(pack)
            if "core" in pack.namespaces and pack.origin_tier != "core":
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_CORE_NAMESPACE_SHADOWED",
                        "error",
                        f"Pack {pack.pack_id} cannot declare the core namespace.",
                    )
                )

            if pack.attestation_kind in {"qst_verified", "signed_pack"}:
                diagnostics.append(
                    _diagnostic(
                        "QST_V2_ATTESTATION_NOT_SELF_TRUSTED",
                        "warning",
                        f"Pack {pack.pack_id} attestation {pack.attestation_kind} is only a claim.",
                    )
                )

            for spec in pack.tokens:
                spec_hash = token_spec_hash_for_spec_v2(spec)
                key = spec.ref_key
                record = RegistryTokenRecord(
                    spec=spec,
                    pack_id=pack.pack_id,
                    pack_origin_tier=pack.origin_tier,
                    token_spec_hash=spec_hash,
                    token_pack_hash=pack_hash,
                )

                if spec.token_ref.namespace == "core" and pack.origin_tier != "core":
                    diagnostics.append(
                        _diagnostic(
                            "QST_V2_CORE_NAMESPACE_SHADOWED",
                            "error",
                            f"Token {spec.token_id} cannot shadow core namespace.",
                        )
                    )

                if spec.attestation_kind in {"qst_verified", "signed_pack"}:
                    diagnostics.append(
                        _diagnostic(
                            "QST_V2_ATTESTATION_NOT_SELF_TRUSTED",
                            "warning",
                            f"Token {spec.token_id} attestation {spec.attestation_kind} is only a claim.",
                        )
                    )

                existing = records.get(key)
                if existing is None:
                    records[key] = record
                    resolution_log.append(f"register {spec.token_id} from {pack.pack_id}")
                    continue

                if existing.token_spec_hash == spec_hash:
                    resolution_log.append(f"dedupe {spec.token_id} from {pack.pack_id}")
                    continue

                override = _choose_project_local_override(existing, record)
                if override is not None:
                    records[key] = override
                    resolution_log.append(f"override {spec.token_id} with {override.pack_id}")
                    diagnostics.append(
                        _diagnostic(
                            "QST_V2_PROJECT_LOCAL_OVERRIDE",
                            "warning",
                            f"Project-local pack {override.pack_id} overrides {spec.token_id}.",
                        )
                    )
                    continue

                diagnostics.append(
                    _diagnostic(
                        "QST_V2_TOKEN_REF_CONFLICT",
                        "error",
                        f"Duplicate token_ref {spec.token_id} has different token_spec_hash.",
                    )
                )

        return cls(
            records=tuple(sorted(records.values(), key=lambda record: record.spec.ref_key)),
            resolution_log=tuple(resolution_log),
            result=ValidationResult(diagnostics=diagnostics),
        )

    def get(self, token_id: str, *, version: int = 1, behavior_version: int = 1) -> RegistryTokenRecord:
        """Resolve a token by token_id/version/behavior_version."""

        for record in self.records:
            if (
                record.spec.token_id == token_id
                and record.spec.version == version
                and record.spec.behavior_version == behavior_version
            ):
                return record
        raise KeyError(f"Token {token_id}/v{version}/bv{behavior_version} not found")


def validate_token_pack_dependencies(
    packs: Iterable[TokenPackManifestV2],
) -> TokenPackDependencyResolution:
    """Validate and resolve TokenPack dependency graph deterministically."""

    sorted_packs = _sort_packs(tuple(packs))
    diagnostics: list[Diagnostic] = []
    by_id: dict[str, list[TokenPackManifestV2]] = defaultdict(list)
    for pack in sorted_packs:
        by_id[pack.pack_id].append(pack)

    selected_edges: dict[str, set[str]] = defaultdict(set)
    for pack in sorted_packs:
        for dependency in pack.dependencies:
            candidates = by_id.get(dependency.pack_id, [])
            if not candidates:
                diagnostics.append(
                    _dependency_diagnostic("QST_V2_TOKEN_PACK_DEP_MISSING", pack, dependency)
                )
                continue
            matching = [candidate for candidate in candidates if dependency.matches(candidate.version)]
            if not matching:
                diagnostics.append(
                    _dependency_diagnostic("QST_V2_TOKEN_PACK_DEP_VERSION_MISMATCH", pack, dependency)
                )
                continue
            selected = _select_dependency_candidate(matching)
            selected_hash = token_pack_hash_for_pack_v2(selected)
            if dependency.token_pack_hash is not None and selected_hash != dependency.token_pack_hash:
                diagnostics.append(
                    _dependency_diagnostic("QST_V2_TOKEN_PACK_DEP_HASH_MISMATCH", pack, dependency)
                )
                continue
            selected_edges[pack.pack_id].add(selected.pack_id)

    for cycle in _find_cycles(selected_edges):
        diagnostics.append(
            _diagnostic(
                "QST_V2_TOKEN_PACK_DEP_CYCLE",
                "error",
                f"TokenPack dependency cycle: {' -> '.join(cycle)}",
            )
        )

    return TokenPackDependencyResolution(
        ordered_packs=tuple(_topological_pack_order(sorted_packs, selected_edges)),
        result=ValidationResult(diagnostics=diagnostics),
    )


def _sort_packs(packs: tuple[TokenPackManifestV2, ...]) -> tuple[TokenPackManifestV2, ...]:
    return tuple(
        sorted(
            packs,
            key=lambda pack: (pack.pack_id, pack.parsed_version, token_pack_hash_for_pack_v2(pack)),
        )
    )


def _select_dependency_candidate(candidates: list[TokenPackManifestV2]) -> TokenPackManifestV2:
    return sorted(
        candidates,
        key=lambda pack: (pack.parsed_version, token_pack_hash_for_pack_v2(pack)),
        reverse=True,
    )[0]


def _choose_project_local_override(
    existing: RegistryTokenRecord,
    candidate: RegistryTokenRecord,
) -> RegistryTokenRecord | None:
    if candidate.spec.token_ref.namespace == "core":
        return None
    if candidate.pack_origin_tier == "user_local":
        return candidate
    if existing.pack_origin_tier == "user_local":
        return existing
    return None


def _find_cycles(edges: dict[str, set[str]]) -> list[tuple[str, ...]]:
    cycles: set[tuple[str, ...]] = set()

    def visit(node: str, path: tuple[str, ...]) -> None:
        if node in path:
            index = path.index(node)
            cycle = (*path[index:], node)
            rotations = [cycle[i:-1] + cycle[:i] + (cycle[i],) for i in range(len(cycle) - 1)]
            cycles.add(min(rotations))
            return
        for next_node in sorted(edges.get(node, set())):
            visit(next_node, (*path, node))

    for node in sorted(edges):
        visit(node, ())
    return sorted(cycles)


def _topological_pack_order(
    packs: tuple[TokenPackManifestV2, ...],
    edges: dict[str, set[str]],
) -> list[TokenPackManifestV2]:
    pack_by_id = {pack.pack_id: pack for pack in packs}
    visited: set[str] = set()
    visiting: set[str] = set()
    ordered_ids: list[str] = []

    def visit(pack_id: str) -> None:
        if pack_id in visited or pack_id in visiting:
            return
        visiting.add(pack_id)
        for dependency_id in sorted(edges.get(pack_id, set())):
            visit(dependency_id)
        visiting.remove(pack_id)
        visited.add(pack_id)
        ordered_ids.append(pack_id)

    for pack in packs:
        visit(pack.pack_id)
    return [pack_by_id[pack_id] for pack_id in ordered_ids if pack_id in pack_by_id]


def _dependency_diagnostic(
    code: str,
    pack: TokenPackManifestV2,
    dependency: TokenPackDependency,
) -> Diagnostic:
    return _diagnostic(
        code,
        "error",
        (
            f"Pack {pack.pack_id} dependency {dependency.pack_id}"
            f" constraint {dependency.version_constraint or '<any>'} failed."
        ),
    )


def _diagnostic(code: str, severity: Severity, message: str) -> Diagnostic:
    return Diagnostic(
        code=code,
        severity=severity,
        phase="token_registry",
        message=message,
    )
