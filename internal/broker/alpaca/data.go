package alpaca

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

// dataHost is the Alpaca market-data API host (same for paper and live).
const dataHost = "https://data.alpaca.markets"

type alpacaAsset struct {
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	AssetClass   string `json:"asset_class"`
	Exchange     string `json:"exchange"`
	Tradable     bool   `json:"tradable"`
	Shortable    bool   `json:"shortable"`
	EasyToBorrow bool   `json:"easy_to_borrow"`
	Status       string `json:"status"`
}

type alpacaClock struct {
	IsOpen    bool   `json:"is_open"`
	NextOpen  string `json:"next_open"`
	NextClose string `json:"next_close"`
}

type alpacaLatestTrade struct {
	Trade struct {
		P float64 `json:"p"`
	} `json:"trade"`
}

type alpacaCorporateAction struct {
	Type     string  `json:"ca_type"`
	ExDate   string  `json:"ex_date"`
	NewRate  float64 `json:"new_rate"`
	OldRate  float64 `json:"old_rate"`
	Cash     float64 `json:"cash"`
	Target   struct {
		Symbol string `json:"symbol"`
	} `json:"target"`
}

// Assets lists active US equity assets from the Alpaca data API.
func (a *AlpacaAdapter) Assets(ctx context.Context) ([]broker.Asset, error) {
	data, err := a.doRequestTo(ctx, dataHost, "GET", "/v2/assets?status=active&asset_class=us_equity", nil)
	if err != nil {
		return nil, err
	}
	var raw []alpacaAsset
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal assets: %w", err)
	}
	out := make([]broker.Asset, 0, len(raw))
	for _, a := range raw {
		if !a.Tradable {
			continue
		}
		out = append(out, broker.Asset{
			Symbol:       a.Symbol,
			Name:         a.Name,
			Class:        a.AssetClass,
			Exchange:     a.Exchange,
			Tradable:     a.Tradable,
			Shortable:    a.Shortable,
			EasyToBorrow: a.EasyToBorrow,
		})
	}
	return out, nil
}

// Clock reports the market clock (broker API).
func (a *AlpacaAdapter) Clock(ctx context.Context) (*broker.MarketClock, error) {
	data, err := a.doRequest(ctx, "GET", "/v2/clock", nil)
	if err != nil {
		return nil, err
	}
	var c alpacaClock
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal clock: %w", err)
	}
	return &broker.MarketClock{
		IsOpen:    c.IsOpen,
		NextOpen:  parseTime(c.NextOpen),
		NextClose: parseTime(c.NextClose),
	}, nil
}

// LatestPrice returns the latest trade price for a symbol (data API).
func (a *AlpacaAdapter) LatestPrice(ctx context.Context, symbol string) (types.Price, error) {
	path := "/v2/stocks/" + url.PathEscape(symbol) + "/trades/latest"
	data, err := a.doRequestTo(ctx, dataHost, "GET", path, nil)
	if err != nil {
		return 0, err
	}
	var t alpacaLatestTrade
	if err := json.Unmarshal(data, &t); err != nil {
		return 0, fmt.Errorf("unmarshal latest trade: %w", err)
	}
	if t.Trade.P <= 0 {
		return 0, fmt.Errorf("no latest trade for %s", symbol)
	}
	return types.PriceFromFloat(t.Trade.P), nil
}

// CorporateActions lists split/dividend announcements for the given symbols
// (broker API, announcement feed).
func (a *AlpacaAdapter) CorporateActions(ctx context.Context, symbols []string) ([]broker.CorporateAction, error) {
	path := "/v1/corporate_actions/announcements?ca_types=split,dividend"
	data, err := a.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var raw []alpacaCorporateAction
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal corporate actions: %w", err)
	}

	want := make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		want[s] = struct{}{}
	}

	out := make([]broker.CorporateAction, 0, len(raw))
	for _, ca := range raw {
		if len(want) > 0 {
			if _, ok := want[ca.Target.Symbol]; !ok {
				continue
			}
		}
		exDate, err := time.Parse("2006-01-02", ca.ExDate)
		if err != nil {
			continue
		}
		action := broker.CorporateAction{
			Symbol: ca.Target.Symbol,
			Date:   exDate,
			Type:   ca.Type,
		}
		switch ca.Type {
		case "split":
			if ca.OldRate > 0 {
				action.SplitRatio = ca.NewRate / ca.OldRate
			}
		case "dividend":
			action.CashDividend = ca.Cash
		}
		out = append(out, action)
	}
	return out, nil
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
