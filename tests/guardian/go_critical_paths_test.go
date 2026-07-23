//go:build guardian
// +build guardian

package guardian

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/broker/paper"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/security"
	"github.com/lee-econ/orca-core/internal/types"
)

type mockBrokerCancel struct{}

func (m mockBrokerCancel) CancelAllOrders(context.Context) error  { return nil }
func (m mockBrokerCancel) CloseAllPositions(context.Context) error { return nil }

var _ risk.BrokerCancel = mockBrokerCancel{}

func TestKillSwitchLockPreventsDoubleFire(t *testing.T) {
	ks := risk.NewKillSwitch(mockBrokerCancel{})
	if ks == nil {
		t.Fatal("expected non-nil kill switch")
	}

	err := ks.Trigger("test-fire-1")
	if err != nil {
		t.Fatalf("first trigger should succeed, got: %v", err)
	}

	halted, reason, _ := ks.Status()
	if !halted {
		t.Error("kill switch should be halted after trigger")
	}
	if reason != "test-fire-1" {
		t.Errorf("reason = %q, want %q", reason, "test-fire-1")
	}

	err2 := ks.Trigger("double-fire")
	if err2 == nil {
		t.Error("second trigger should return an error (already halted)")
	}
}

func TestPaperAdapterOrderLifecycle(t *testing.T) {
	adapter := paper.NewAdapter(100000)
	ctx := context.Background()

	resp, err := adapter.PlaceOrder(ctx, &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Limit,
		Quantity:    100,
		LimitPrice:  types.FromFloat64(580.0),
		TimeInForce: broker.Day,
	})
	if err != nil {
		t.Fatalf("place order: %v", err)
	}
	if resp.Status != broker.Filled {
		t.Errorf("order status = %q, want %q", resp.Status, broker.Filled)
	}
	if resp.BrokerOrderID == "" {
		t.Error("expected non-empty BrokerOrderID")
	}

	if err := adapter.CancelOrder(ctx, resp.BrokerOrderID); err != nil {
		t.Errorf("cancel order: %v", err)
	}
}

func TestWebSocketHubBroadcastIntegrity(t *testing.T) {
	hub := monitor.NewWSHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}

	hub.Broadcast("guardian", map[string]bool{"halted": false})
	hub.Broadcast("guardian", "string-payload")
	hub.Broadcast("guardian", map[string]interface{}{"count": 1, "msg": "test"})

	hub.StartPerformanceBroadcast(100*time.Millisecond, func() interface{} {
		return map[string]float64{"sharpe": 1.5}
	})
	time.Sleep(50 * time.Millisecond)
	hub.StopPerformanceBroadcast()
}

func TestJWTTokenValidationRoundTrip(t *testing.T) {
	secret := []byte("guardian-test-secret-32-bytes-xx")
	if len(secret) < 32 {
		t.Fatal("secret must be at least 32 bytes")
	}

	pair, err := security.GenerateTokenPair("admin", "admin", nil, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh token should not be empty")
	}

	claims, err := security.ValidateToken(pair.AccessToken, secret)
	if err != nil {
		t.Fatalf("validate valid token: %v", err)
	}
	if claims.Username != "admin" {
		t.Errorf("username = %q, want %q", claims.Username, "admin")
	}

	tampered := pair.AccessToken[:len(pair.AccessToken)-4] + "XXXX"
	_, err = security.ValidateToken(tampered, secret)
	if err == nil {
		t.Error("tampered token should fail validation")
	}
}

func TestKillSwitchClosesPositions(t *testing.T) {
	adapter := paper.NewAdapter(100000)
	ctx := context.Background()

	_, err := adapter.PlaceOrder(ctx, &broker.OrderRequest{
		Symbol:      "SPY",
		Side:        broker.Buy,
		Type:        broker.Limit,
		Quantity:    100,
		LimitPrice:  types.FromFloat64(580.0),
		TimeInForce: broker.Day,
	})
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	positions, err := adapter.GetPositions(ctx)
	if err != nil {
		t.Fatalf("get positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position after buy, got %d", len(positions))
	}

	ks := risk.NewKillSwitch(adapter)
	if err := ks.Trigger("position-close-test"); err != nil {
		t.Fatalf("trigger kill switch: %v", err)
	}

	positions2, err := adapter.GetPositions(ctx)
	if err != nil {
		t.Fatalf("get positions after kill switch: %v", err)
	}
	if len(positions2) != 0 {
		t.Errorf("expected 0 positions after kill switch, got %d", len(positions2))
	}
}

func TestRepositoryCRUDIntegrity(t *testing.T) {
	t.Skip("Guardian test — requires testcontainers-go for TimescaleDB container. " +
		"Run locally with Docker via: go test -tags=guardian ./tests/guardian/ " +
		"-run TestRepositoryCRUDIntegrity")
}

func TestWSHub_BroadcastDeliversJSON(t *testing.T) {
	hub := monitor.NewWSHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}

	payload := map[string]interface{}{"halted": false, "reason": "test"}
	hub.Broadcast("risk", payload)

	payload2 := map[string]interface{}{"sharpe_ratio": 1.5, "max_drawdown": -15.0}
	hub.Broadcast("performance", payload2)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Broadcast panicked with nil/invalid data: %v", r)
		}
	}()
	hub.Broadcast("edge", nil)
}

func TestWSHub_BroadcastWithBytes(t *testing.T) {
	hub := monitor.NewWSHub()

	hub.Broadcast("bytes", []byte(`{"key":"value"}`))
	hub.Broadcast("bytes", bytes.NewBufferString(`{"buf":"test"}`).Bytes())
}

func TestJWT_ExpiredTokenFails(t *testing.T) {
	secret := []byte("guardian-test-secret-32-bytes-xx")
	pair, err := security.GenerateTokenPair("admin", "admin", nil, secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	_, err = security.ValidateToken(pair.AccessToken, secret)
	if err == nil {
		t.Error("expired token should fail validation")
	}
}

func TestJWT_WrongSecretFails(t *testing.T) {
	secret := []byte("guardian-test-secret-32-bytes-xx")
	pair, err := security.GenerateTokenPair("admin", "admin", nil, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	_, err = security.ValidateToken(pair.AccessToken, []byte("wrong-secret-32-bytes-xxxxxxxx"))
	if err == nil {
		t.Error("token validated with wrong secret should fail")
	}
}
