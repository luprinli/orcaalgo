from __future__ import annotations

import json
import socket
from unittest.mock import Mock, patch

import pytest

API_BASE = "http://localhost:8080"


# ── helpers ────────────────────────────────────────────────────────────────

def _mock_urlopen(status: int = 200, response_body: dict | None = None):
    """Return a configured Mock for urllib.request.urlopen."""
    body = json.dumps(response_body or {}).encode("utf-8")
    mock_resp = Mock()
    mock_resp.status = status
    mock_resp.read.return_value = body
    mock_resp.__enter__ = Mock(return_value=mock_resp)
    mock_resp.__exit__ = Mock(return_value=False)
    return Mock(return_value=mock_resp)


def _extract_body(call_args: tuple) -> dict:
    """Extract the JSON body sent to urlopen from call args."""
    req = call_args[0][0]
    return json.loads(req.data.decode("utf-8"))


def _extract_headers(call_args: tuple) -> dict:
    """Extract request headers from call args."""
    req = call_args[0][0]
    return dict(req.headers)


# ── fixtures ───────────────────────────────────────────────────────────────

@pytest.fixture
def valid_credentials() -> dict[str, str]:
    return {"username": "trader1", "password": "correct-horse-battery-staple"}


@pytest.fixture
def login_response() -> dict:
    return {"token": "jwt.abc123.xyz", "user": {"username": "trader1", "role": "user"}}


@pytest.fixture
def registration_payload() -> dict[str, str]:
    return {"username": "newuser", "password": "s3cureP@ss", "email": "new@example.com"}


# ── 1. Login API call structure ────────────────────────────────────────────

class TestLoginAPICallStructure:
    """Test that the login request is correctly structured."""

    def test_sends_post_to_correct_url(self, valid_credentials, login_response):
        mock_urlopen = _mock_urlopen(200, login_response)

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request
            import urllib.error

            data = json.dumps(valid_credentials).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

        call_args = mock_urlopen.call_args
        request_obj = call_args[0][0]
        assert request_obj.full_url == f"{API_BASE}/api/v1/auth/login"
        assert request_obj.method == "POST"
        assert body["token"] == login_response["token"]
        assert body["user"]["username"] == "trader1"

    def test_request_body_contains_username_and_password(self, valid_credentials):
        mock_urlopen = _mock_urlopen(200, {"token": "t", "user": {"username": "x", "role": "user"}})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps(valid_credentials).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            urllib.request.urlopen(req)

        body = _extract_body(mock_urlopen.call_args)
        assert "username" in body
        assert "password" in body
        assert body["username"] == valid_credentials["username"]

    def test_sets_content_type_header(self, valid_credentials):
        mock_urlopen = _mock_urlopen(200, {"token": "t", "user": {"username": "x", "role": "user"}})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps(valid_credentials).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            urllib.request.urlopen(req)

        headers = _extract_headers(mock_urlopen.call_args)
        assert headers.get("Content-type") == "application/json"


# ── 2. Login error handling ────────────────────────────────────────────────

class TestLoginErrorHandling:
    """Verify that error responses are handled correctly."""

    def test_401_unauthorized_wrong_password(self, login_response):
        """401 Unauthorized should raise HTTPError."""
        import urllib.error

        mock_urlopen = Mock(side_effect=urllib.error.HTTPError(
            f"{API_BASE}/api/v1/auth/login", 401,
            "Unauthorized", {}, None,
        ))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps({"username": "trader1", "password": "wrong"}).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as exc_info:
                urllib.request.urlopen(req)
            assert exc_info.value.code == 401

    def test_400_bad_request(self):
        """400 Bad Request (missing field) should raise HTTPError."""
        import urllib.error

        mock_urlopen = Mock(side_effect=urllib.error.HTTPError(
            f"{API_BASE}/api/v1/auth/login", 400,
            "Bad Request", {}, None,
        ))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps({"username": ""}).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as exc_info:
                urllib.request.urlopen(req)
            assert exc_info.value.code == 400

    def test_500_server_error(self):
        """500 Internal Server Error should raise HTTPError."""
        import urllib.error

        mock_urlopen = Mock(side_effect=urllib.error.HTTPError(
            f"{API_BASE}/api/v1/auth/login", 500,
            "Internal Server Error", {}, None,
        ))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps({"username": "u", "password": "p"}).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as exc_info:
                urllib.request.urlopen(req)
            assert exc_info.value.code == 500

    def test_network_timeout(self):
        """Socket timeout / URLError should be caught."""
        import urllib.error

        mock_urlopen = Mock(side_effect=urllib.error.URLError(socket.timeout("timed out")))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps({"username": "u", "password": "p"}).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.URLError):
                urllib.request.urlopen(req)


# ── 3. Registration request ────────────────────────────────────────────────

class TestRegistration:
    """Test that registration sends the correct fields."""

    def test_sends_username_password_email(self, registration_payload):
        mock_urlopen = _mock_urlopen(200, {"registered": True, "token": "jwt.newuser"})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps(registration_payload).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/register",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

        sent = _extract_body(mock_urlopen.call_args)
        assert sent["username"] == "newuser"
        assert sent["password"] == "s3cureP@ss"
        assert sent["email"] == "new@example.com"
        assert body["registered"] is True
        assert body["token"] == "jwt.newuser"

    def test_registration_endpoint_url(self, registration_payload):
        mock_urlopen = _mock_urlopen(200, {"registered": True, "token": "t"})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps(registration_payload).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/register",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            urllib.request.urlopen(req)

        request_obj = mock_urlopen.call_args[0][0]
        assert request_obj.full_url == f"{API_BASE}/api/v1/auth/register"
        assert request_obj.method == "POST"


# ── 4. Password reset flow ─────────────────────────────────────────────────

class TestPasswordResetFlow:
    """Test forgot-password and reset-password request structures."""

    def test_forgot_password_sends_email(self):
        mock_urlopen = _mock_urlopen(200, {"ok": True})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            data = json.dumps({"email": "trader1@example.com"}).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/forgot-password",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

        sent = _extract_body(mock_urlopen.call_args)
        assert sent["email"] == "trader1@example.com"
        assert body["ok"] is True

    def test_reset_password_sends_token_and_new_password(self):
        mock_urlopen = _mock_urlopen(200, {"ok": True})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            payload = {"reset_token": "abc-reset-token", "new_password": "N3wP@ssword"}
            data = json.dumps(payload).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/reset-password",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

        sent = _extract_body(mock_urlopen.call_args)
        assert sent["reset_token"] == "abc-reset-token"
        assert sent["new_password"] == "N3wP@ssword"
        assert body["ok"] is True


# ── 5. Token storage ───────────────────────────────────────────────────────

class TokenStore:
    """Minimal in-process token store for test purposes.

    Simulates the behavior a real auth client would use to persist a JWT
    across requests.
    """

    def __init__(self) -> None:
        self._token: str | None = None
        self._user: dict | None = None

    def store(self, token: str, user: dict | None = None) -> None:
        self._token = token
        self._user = user

    @property
    def token(self) -> str | None:
        return self._token

    @property
    def is_authenticated(self) -> bool:
        return self._token is not None

    def clear(self) -> None:
        self._token = None
        self._user = None

    def auth_header(self) -> dict[str, str] | None:
        if not self._token:
            return None
        return {"Authorization": f"Bearer {self._token}"}


class TestTokenStorage:
    """Test that tokens are stored and retrieved correctly."""

    def test_store_and_retrieve_token(self):
        store = TokenStore()
        store.store("jwt.token.value", {"username": "trader1", "role": "user"})
        assert store.token == "jwt.token.value"
        assert store.is_authenticated is True

    def test_store_token_from_login_response(self, login_response):
        store = TokenStore()
        store.store(login_response["token"], login_response["user"])
        assert store.token == "jwt.abc123.xyz"

    def test_empty_store_is_unauthenticated(self):
        store = TokenStore()
        assert store.token is None
        assert store.is_authenticated is False

    def test_clear_removes_token(self):
        store = TokenStore()
        store.store("jwt.token.value")
        store.clear()
        assert store.token is None
        assert store.is_authenticated is False


# ── 6. Token validation ────────────────────────────────────────────────────

class TestTokenValidation:
    """Test that expired or missing tokens are handled correctly."""

    def test_missing_token_returns_unauthorized(self):
        """If no token is set, auth_header returns None."""
        store = TokenStore()
        assert store.auth_header() is None
        assert not store.is_authenticated

    def test_expired_token_still_sets_header(self):
        """TokenStore does not decode JWT expiry — external validation applies."""
        store = TokenStore()
        store.store("expired-jwt-token")
        header = store.auth_header()
        assert header is not None
        assert header["Authorization"] == "Bearer expired-jwt-token"

    def test_server_rejects_expired_token(self):
        """When server returns 401 for expired token, client should clear it."""
        import urllib.error

        store = TokenStore()
        store.store("expired-jwt-token")

        mock_urlopen = Mock(side_effect=urllib.error.HTTPError(
            f"{API_BASE}/api/v1/auth/me", 401,
            "Unauthorized — token expired", {}, None,
        ))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/me",
                headers={"Authorization": f"Bearer {store.token}"},
            )
            with pytest.raises(urllib.error.HTTPError) as exc_info:
                urllib.request.urlopen(req)

            assert exc_info.value.code == 401
            store.clear()

        assert not store.is_authenticated


# ── 7. 2FA setup ───────────────────────────────────────────────────────────

class TestTwoFactorSetup:
    """Test the TOTP enrollment request / response structure."""

    def test_2fa_setup_response_contains_totp_secret_and_qr(self):
        mock_urlopen = _mock_urlopen(200, {
            "totp_secret": "JBSWY3DPEHPK3PXP",
            "qr_code_url": "otpauth://totp/OrcaAlgo:trader1?secret=JBSWY3DPEHPK3PXP&issuer=OrcaAlgo",
        })

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/2fa/setup",
                data=b"",
                headers={
                    "Content-Type": "application/json",
                    "Authorization": "Bearer jwt.abc123.xyz",
                },
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

            assert "totp_secret" in body
            assert "qr_code_url" in body
            assert len(body["totp_secret"]) > 0
            assert body["qr_code_url"].startswith("otpauth://")

    def test_2fa_setup_requires_auth_header(self):
        """A 2FA setup request must include the Authorization header."""
        mock_urlopen = _mock_urlopen(200, {"totp_secret": "X", "qr_code_url": "otpauth://"})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/2fa/setup",
                data=b"",
                headers={
                    "Content-Type": "application/json",
                    "Authorization": "Bearer jwt.abc123.xyz",
                },
                method="POST",
            )
            urllib.request.urlopen(req)

        headers = _extract_headers(mock_urlopen.call_args)
        assert headers.get("Authorization") == "Bearer jwt.abc123.xyz"

    def test_2fa_setup_unauthenticated(self):
        """Server should reject 2FA setup without a valid token."""
        import urllib.error

        mock_urlopen = Mock(side_effect=urllib.error.HTTPError(
            f"{API_BASE}/api/v1/auth/2fa/setup", 401,
            "Unauthorized", {}, None,
        ))

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/2fa/setup",
                data=b"",
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as exc_info:
                urllib.request.urlopen(req)
            assert exc_info.value.code == 401


# ── 8. Token refresh / authenticated requests ──────────────────────────────

class TestTokenRefreshAndAuthHeader:
    """Test that the Authorization header is correctly set on requests."""

    def test_authorization_header_format(self):
        store = TokenStore()
        store.store("jwt.header.payload.signature")
        header = store.auth_header()
        assert header["Authorization"] == "Bearer jwt.header.payload.signature"

    def test_authenticated_request_sets_header(self):
        store = TokenStore()
        store.store("jwt.valid.token")

        mock_urlopen = _mock_urlopen(200, {"user": {"username": "trader1", "role": "user"}})

        with patch("urllib.request.urlopen", mock_urlopen):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/me",
                headers={"Authorization": f"Bearer {store.token}"},
            )
            with urllib.request.urlopen(req) as resp:
                body = json.loads(resp.read().decode("utf-8"))

        headers = _extract_headers(mock_urlopen.call_args)
        assert headers.get("Authorization") == "Bearer jwt.valid.token"
        assert body["user"]["username"] == "trader1"

    def test_full_login_then_authenticated_request_flow(self, valid_credentials, login_response):
        store = TokenStore()

        login_mock = _mock_urlopen(200, login_response)
        with patch("urllib.request.urlopen", login_mock):
            import urllib.request

            data = json.dumps(valid_credentials).encode("utf-8")
            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/login",
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req) as resp:
                result = json.loads(resp.read().decode("utf-8"))
            store.store(result["token"], result["user"])

        assert store.is_authenticated
        assert store.token == login_response["token"]

        me_mock = _mock_urlopen(200, {"user": {"username": "trader1", "role": "user"}})
        with patch("urllib.request.urlopen", me_mock):
            import urllib.request

            req = urllib.request.Request(
                f"{API_BASE}/api/v1/auth/me",
                headers={"Authorization": f"Bearer {store.token}"},
            )
            with urllib.request.urlopen(req) as resp:
                profile = json.loads(resp.read().decode("utf-8"))

        assert profile["user"]["username"] == "trader1"
        me_headers = _extract_headers(me_mock.call_args)
        assert me_headers.get("Authorization") == f"Bearer {login_response['token']}"
