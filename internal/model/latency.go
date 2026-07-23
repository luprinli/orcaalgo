package model

import "time"

type LatencyModel interface {
	EntryLatency(ts time.Time, price uint64, quantity uint64, side string) time.Duration
	ResponseLatency(ts time.Time, price uint64, side string) time.Duration
}

type ConstantLatency struct {
	Entry    time.Duration
	Response time.Duration
}

func (c ConstantLatency) EntryLatency(ts time.Time, price uint64, quantity uint64, side string) time.Duration {
	return c.Entry
}

func (c ConstantLatency) ResponseLatency(ts time.Time, price uint64, side string) time.Duration {
	return c.Response
}

type ZeroLatency struct{}

func (z ZeroLatency) EntryLatency(ts time.Time, price uint64, quantity uint64, side string) time.Duration {
	return 0
}

func (z ZeroLatency) ResponseLatency(ts time.Time, price uint64, side string) time.Duration {
	return 0
}
