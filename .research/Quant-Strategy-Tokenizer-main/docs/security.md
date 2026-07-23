# Security Boundary

QST separates record validation from code execution.

## Custom Tokens

Custom-token integrity verification checks TokenSpec, TokenPack, implementation references, runtime environment metadata, dependency hashes, and audit material without importing or executing user code. It does not call entry-point loaders, inspect custom modules, or perform dynamic package introspection.

Execution requires:

1. Integrity verification.
2. Authorization under the selected profile.
3. A local approval record with explicit risk acknowledgement and token allowance.
4. A short-lived execution grant bound to token, pack, implementation, runtime, profile, approval, and run id.
5. Output validation against declared TokenSpec ports and numeric policy.

There is no sandbox here. Approved custom-token execution runs local Python code.

## Non-Portable Trust

Approval records and execution grants are local security state. They are not canonical strategy material and are not portable trust. Receiving a package or source tree never implies approval to execute code.

## External Systems

QST does not provide broker, exchange, custody, order-routing, or production trading controls. Integrations must enforce their own risk checks, credentials, throttles, and operational approvals.