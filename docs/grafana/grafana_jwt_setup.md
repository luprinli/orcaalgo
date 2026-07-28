# Grafana JWT OAuth Setup

## Overview

Grafana is now configured to authenticate using Orca's JWT tokens via the `auth.jwt` module.  
This allows traders to log into Grafana with the same credentials they use for the React SPA,
with role-based access control.

## How It Works

1. The React frontend authenticates against the Go API and receives a JWT.
2. When the user navigates to an embedded Grafana iframe (InfrastructureTab) or opens
   Grafana directly, the JWT is passed in the `Authorization: Bearer <token>` header.
3. Grafana's `auth.jwt` middleware validates the JWT using the shared secret
   (mounted as `/etc/grafana/jwt_secret.txt`).
4. The `role` claim determines Grafana access:
   - `admin` → Grafana Admin (full dashboard editing, user management)
   - `trader` → Grafana Viewer (read-only dashboard access)
5. Users are auto-created on first access (auto_sign_up).

## Files

| File | Purpose |
|------|---------|
| `configs/grafana/grafana.ini` | Grafana server configuration with `[auth.jwt]` section |
| `configs/grafana/jwt_secret.txt` | Shared JWT secret (same as `ORCA_JWT_SECRET`) |
| `docker-compose.yml` | Mounts both files into the Grafana container |

## JWT Claims Expected

Grafana expects the following fields in the JWT payload:

```json
{
  "sub": "user@example.com",
  "role": "trader",
  "exp": 1720000000
}
```

The Go server's JWT middleware (`internal/api/middleware/middleware.go`) must include the `role`
claim.  If the claim is missing, the user defaults to the Grafana `Viewer` role.

## Verification

After starting the stack:

```bash
# 1. Obtain a valid JWT from the API (example using curl)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@orca.local","password":"admin123"}' | jq -r '.token')

# 2. Use the token to access Grafana
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3000/api/user | jq .
```

## Troubleshooting

- **"User not found":** The `auto_sign_up` setting must be `true`.
- **"Invalid token":** Verify `jwt_secret.txt` matches `ORCA_JWT_SECRET`.
- **Role not applied:** Check the JWT payload includes a `role` claim.
- **CORS issues:** Embedded iframes require `allow_embedding = true` in `[security]`.

## Security Notes

- The `jwt_secret.txt` file contains the JWT secret in plaintext — ensure file permissions
  restrict access (chmod 600).
- In production, rotate the default secret (`dev-secret-change-in-production`).
- Consider using JWKS (JSON Web Key Set) for key rotation instead of a shared secret file.
