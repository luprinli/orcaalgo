package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type RestClient struct {
	baseURL    string
	httpClient *http.Client
	accountID  string
	mu         sync.Mutex
}

func NewRestClient(baseURL, accountID string, timeout time.Duration) *RestClient {
	return &RestClient{
		baseURL:    baseURL,
		accountID:  accountID,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *RestClient) authenticate(ctx context.Context) error {
	resp, err := c.request(ctx, http.MethodPost, "/iserver/auth/ssodh/init", nil)
	if err != nil {
		return fmt.Errorf("ibkr: auth init: %w", err)
	}
	var authResp struct {
		Authenticated bool   `json:"authenticated"`
		Message       string `json:"message"`
	}
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return fmt.Errorf("ibkr: parse auth response: %w", err)
	}
	if !authResp.Authenticated {
		return fmt.Errorf("ibkr: authentication failed: %s", authResp.Message)
	}
	return nil
}

func (c *RestClient) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ibkr: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("ibkr: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.mu.Lock()
	defer c.mu.Unlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ibkr: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ibkr: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("ibkr: %s (status %d)", errResp.Error, resp.StatusCode)
		}
		return nil, fmt.Errorf("ibkr: request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type ibkrOrderRequest struct {
	AcctID     string      `json:"acctId"`
	Conid      int         `json:"conid"`
	SecType    string      `json:"secType"`
	OrderType  string      `json:"orderType"`
	Side       string      `json:"side"`
	Quantity   float64     `json:"quantity"`
	Price      types.Price `json:"price,omitempty"`
	AuxPrice   types.Price `json:"auxPrice,omitempty"`
	Tif        string      `json:"tif"`
	OutsideRTH bool        `json:"outsideRTH"`
}

type ibkrOrderResponse struct {
	OrderID      string `json:"order_id"`
	OrderStatus  string `json:"order_status"`
	LocalOrderID string `json:"local_order_id"`
}

type ibkrPosition struct {
	AcctID        string  `json:"acctId"`
	Conid         int     `json:"conid"`
	ContractDesc  string  `json:"contractDesc"`
	Position      float64 `json:"position"`
	MktPrice      float64 `json:"mktPrice"`
	MktValue      float64 `json:"mktValue"`
	AvgCost       float64 `json:"avgCost"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
}

type ibkrPnlResponse struct {
	Total struct {
		ROW []struct {
			DailyPnL       float64 `json:"dailyPnL"`
			UnrealizedPnL  float64 `json:"unrealizedPnL"`
			RealizedPnL    float64 `json:"realizedPnL"`
			NetLiquidation float64 `json:"netLiquidation"`
		} `json:"ROW"`
	} `json:"Total"`
}
