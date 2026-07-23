# Kalman Custom Token Example

This is an example custom TokenPack.

It is not part of the QST core token vocabulary. It is not a built-in QST token.
It demonstrates the custom token verify / approve / execute boundary using a
deterministic local Python `python_entrypoint`.

The package executes local Python source when approved. Verification inspects
metadata and hashes only; it does not import or execute the source module.
