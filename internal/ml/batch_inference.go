package ml

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// BatchInferrer applies meta-labeling to multiple signals efficiently using:
// 1. Threshold skipping — skip ML for extreme confidence signals
// 2. Cache lookup — reuse previous predictions for unchanged features
// 3. Sequential inference — calls Predictor for remaining signals
type BatchInferrer struct {
	predictor Predictor
	cache     map[string]cacheEntry
	mu        sync.RWMutex
	cfg       MetaLabelerConfig
	metrics   *BatchMetrics
}

type cacheEntry struct {
	pWin   float64
	expiry time.Time
}

// BatchMetrics tracks inference performance.
type BatchMetrics struct {
	TotalCalls      int64
	CacheHits       int64
	SkippedLow      int64
	SkippedHigh     int64
	TotalInferences int64
	InferenceErrors int64
}

// cacheTTL is the duration a cached prediction remains valid.
const cacheTTL = 5 * time.Minute

// maxCacheEntries limits cache growth.
const maxCacheEntries = 10000

// NewBatchInferrer creates a new batch inferrer with caching.
func NewBatchInferrer(predictor Predictor, cfg MetaLabelerConfig) *BatchInferrer {
	return &BatchInferrer{
		predictor: predictor,
		cache:     make(map[string]cacheEntry),
		cfg:       cfg,
		metrics:   &BatchMetrics{},
	}
}

// Evaluate evaluates a single signal with caching and threshold skipping.
func (bi *BatchInferrer) Evaluate(
	features []float32,
	signalConfidence float64,
) MetaLabelingResult {
	bi.mu.Lock()
	bi.metrics.TotalCalls++
	bi.mu.Unlock()

	// Layer 1: Threshold skipping
	if signalConfidence <= bi.cfg.ExtremeLow {
		bi.mu.Lock()
		bi.metrics.SkippedLow++
		bi.mu.Unlock()
		return MetaLabelingResult{
			PWin:      0.0,
			Threshold: bi.cfg.WinThreshold,
			Accepted:  false,
			Reason: fmt.Sprintf("skipped_extreme_low: confidence=%.3f",
				signalConfidence),
		}
	}
	if signalConfidence >= bi.cfg.ExtremeHigh {
		bi.mu.Lock()
		bi.metrics.SkippedHigh++
		bi.mu.Unlock()
		return MetaLabelingResult{
			PWin:      1.0,
			Threshold: bi.cfg.WinThreshold,
			Accepted:  true,
			Reason: fmt.Sprintf("skipped_extreme_high: confidence=%.3f",
				signalConfidence),
		}
	}

	// Layer 2: Cache lookup
	cacheKey := makeCacheKey(features)
	bi.mu.RLock()
	if entry, ok := bi.cache[cacheKey]; ok && time.Now().Before(entry.expiry) {
		bi.mu.RUnlock()
		bi.mu.Lock()
		bi.metrics.CacheHits++
		bi.mu.Unlock()
		accepted := entry.pWin >= bi.cfg.WinThreshold
		return MetaLabelingResult{
			PWin:      entry.pWin,
			Threshold: bi.cfg.WinThreshold,
			Accepted:  accepted,
			Reason:    fmt.Sprintf("cache_hit: p_win=%.3f", entry.pWin),
		}
	}
	bi.mu.RUnlock()

	// Layer 3: Inference
	bi.mu.Lock()
	bi.metrics.TotalInferences++
	bi.mu.Unlock()

	pWin, err := bi.predictor.Predict(features)
	if err != nil {
		bi.mu.Lock()
		bi.metrics.InferenceErrors++
		bi.mu.Unlock()
		return MetaLabelingResult{
			PWin:      1.0,
			Threshold: bi.cfg.WinThreshold,
			Accepted:  true,
			Reason:    fmt.Sprintf("inference_error: %v", err),
		}
	}

	// Store in cache with eviction if full
	bi.mu.Lock()
	if len(bi.cache) >= maxCacheEntries {
		bi.evictOldest()
	}
	bi.cache[cacheKey] = cacheEntry{pWin: pWin, expiry: time.Now().Add(cacheTTL)}
	bi.mu.Unlock()

	accepted := pWin >= bi.cfg.WinThreshold
	return MetaLabelingResult{
		PWin:      pWin,
		Threshold: bi.cfg.WinThreshold,
		Accepted:  accepted,
		Reason:    fmt.Sprintf("inference: p_win=%.3f", pWin),
	}
}

// EvaluateBatch evaluates multiple signals sequentially.
func (bi *BatchInferrer) EvaluateBatch(
	featureSets [][]float32,
	signalConfidences []float64,
) []MetaLabelingResult {
	results := make([]MetaLabelingResult, len(featureSets))
	for i := range featureSets {
		results[i] = bi.Evaluate(featureSets[i], signalConfidences[i])
	}
	return results
}

// Metrics returns a copy of the current batch metrics.
func (bi *BatchInferrer) Metrics() BatchMetrics {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return *bi.metrics
}

// InvalidateCache clears all cached predictions.
func (bi *BatchInferrer) InvalidateCache() {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.cache = make(map[string]cacheEntry)
}

// evictOldest removes one entry from cache (must hold write lock).
func (bi *BatchInferrer) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range bi.cache {
		if oldestKey == "" || v.expiry.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiry
		}
	}
	if oldestKey != "" {
		delete(bi.cache, oldestKey)
	}
}

func makeCacheKey(features []float32) string {
	h := sha256.New()
	for _, f := range features {
		h.Write([]byte(fmt.Sprintf("%.6f", f)))
	}
	bucket := time.Now().Unix() / 300
	h.Write([]byte(fmt.Sprintf("%d", bucket)))
	return hex.EncodeToString(h.Sum(nil))
}
