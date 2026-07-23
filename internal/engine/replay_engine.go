package engine

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lee-econ/orca-core/internal/strategy"
)

type ReplayConfig struct {
	TicksDir       string
	SpeedMultiplier float64
	BrokerMode     string
	SymbolMap      map[string]uint32
	EventDriven    bool
}

type SyntheticTick struct {
	TimestampMS int64   `json:"timestamp_ms"`
	Price       float64 `json:"price"`
	Bid         float64 `json:"bid"`
	Ask         float64 `json:"ask"`
	Volume      int     `json:"volume"`
	Symbol      string  `json:"symbol"`
}

type ReplayEngine struct {
	cfg    ReplayConfig
	engine *LiveEngine
	logger *slog.Logger
}

func NewReplayEngine(engine *LiveEngine, cfg ReplayConfig, logger *slog.Logger) *ReplayEngine {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if cfg.SpeedMultiplier <= 0 {
		cfg.SpeedMultiplier = 1.0
	}
	return &ReplayEngine{
		cfg:    cfg,
		engine: engine,
		logger: logger,
	}
}

func (r *ReplayEngine) LoadTicks(symbol string, generationID string) ([]SyntheticTick, error) {
	var jsonlPath string
	if generationID != "" {
		jsonlPath = filepath.Join(r.cfg.TicksDir, symbol, generationID, "ticks.jsonl")
	} else {
		entries, err := os.ReadDir(filepath.Join(r.cfg.TicksDir, symbol))
		if err != nil {
			return nil, fmt.Errorf("read ticks dir %s: %w", filepath.Join(r.cfg.TicksDir, symbol), err)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("no generation dirs found for %s", symbol)
		}
		jsonlPath = filepath.Join(r.cfg.TicksDir, symbol, entries[0].Name(), "ticks.jsonl")
	}

	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", jsonlPath, err)
	}
	defer f.Close()

	var ticks []SyntheticTick
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var t SyntheticTick
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			r.logger.Warn("skip malformed tick line", "err", err)
			continue
		}
		ticks = append(ticks, t)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", jsonlPath, err)
	}
	return ticks, nil
}

func (r *ReplayEngine) Replay(ticks []SyntheticTick) ([]*strategy.Signal, error) {
	if len(ticks) == 0 {
		return nil, errors.New("no ticks to replay")
	}

	var allSignals []*strategy.Signal

	startReal := time.Now()
	startTickMS := ticks[0].TimestampMS
	speed := r.cfg.SpeedMultiplier

	for i, t := range ticks {
		if r.cfg.EventDriven && i > 0 && t.TimestampMS == ticks[i-1].TimestampMS {
			// same-timestamp events: process without sleeping
		} else {
			tickElapsedMS := t.TimestampMS - startTickMS
			targetElapsed := time.Duration(float64(tickElapsedMS) / speed) * time.Millisecond
			targetTime := startReal.Add(targetElapsed)
			wait := time.Until(targetTime)
			if wait > 0 {
				time.Sleep(wait)
			}
		}

		symbolID, ok := r.cfg.SymbolMap[t.Symbol]
		if !ok {
			if r.cfg.SymbolMap == nil {
				r.cfg.SymbolMap = make(map[string]uint32)
			}
			symbolID = uint32(len(r.cfg.SymbolMap) + 1)
			r.cfg.SymbolMap[t.Symbol] = symbolID
		}

		priceRaw := uint64(t.Price * 100_000)
		volumeRaw := uint64(t.Volume)

		signals := r.engine.ProcessTick(
			symbolID,
			priceRaw,
			volumeRaw,
			t.TimestampMS*1_000_000,
		)

		if len(signals) > 0 {
			allSignals = append(allSignals, signals...)
			for _, sig := range signals {
				r.logger.Info("signal",
					"symbol", t.Symbol,
					"side", sig.Side,
					"quantity", sig.Quantity,
					"price", t.Price,
					"tick_index", i,
				)
			}
		}

		if i > 0 && i%100_000 == 0 {
			r.logger.Info("replay progress", "ticks_replayed", i, "total", len(ticks))
		}
	}

	elapsed := time.Since(startReal)
	r.logger.Info("replay complete",
		"ticks", len(ticks),
		"duration", elapsed,
		"speed", speed,
		"signals", len(allSignals),
	)
	return allSignals, nil
}

func (r *ReplayEngine) LoadAndReplay(symbol string, generationID string) ([]*strategy.Signal, error) {
	ticks, err := r.LoadTicks(symbol, generationID)
	if err != nil {
		return nil, err
	}
	return r.Replay(ticks)
}
