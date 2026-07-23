"""Token System v2 custom token runtime service.

WP9 keeps custom token execution behind explicit integrity, authorization, and
execution-grant checks. Integrity verification never imports or executes custom
token code.
"""

from qst.custom_runtime.audit import (
    AuditRecord,
    audit_chain_hash_for_records,
)
from qst.custom_runtime.implementation import (
    ImplementationRef,
    ReproducibilityLevel,
    RuntimeEnvironmentRef,
    implementation_ref_hash_for_ref,
    runtime_environment_ref_current,
    runtime_environment_ref_hash_for_ref,
)
from qst.custom_runtime.models import (
    ApprovalRecord,
    ApprovalRequest,
    AuthorizationStatus,
    ExecutionGrant,
    TokenAuthorizationResult,
    TokenExecutionResult,
    TokenIntegrityResult,
    TokenRuntimeContext,
    TokenVerifyReport,
    approval_record_hash,
)
from qst.custom_runtime.pack_io import load_token_pack
from qst.custom_runtime.service import (
    ApprovalStore,
    TokenRuntimeService,
    approve_token_pack,
    check_authorization,
    execute_custom_token,
    verify_integrity,
)

__all__ = [
    "ApprovalRecord",
    "ApprovalRequest",
    "ApprovalStore",
    "AuditRecord",
    "AuthorizationStatus",
    "ExecutionGrant",
    "ImplementationRef",
    "ReproducibilityLevel",
    "RuntimeEnvironmentRef",
    "TokenAuthorizationResult",
    "TokenExecutionResult",
    "TokenIntegrityResult",
    "TokenRuntimeContext",
    "TokenRuntimeService",
    "TokenVerifyReport",
    "approval_record_hash",
    "approve_token_pack",
    "audit_chain_hash_for_records",
    "check_authorization",
    "execute_custom_token",
    "implementation_ref_hash_for_ref",
    "load_token_pack",
    "runtime_environment_ref_current",
    "runtime_environment_ref_hash_for_ref",
    "verify_integrity",
]
