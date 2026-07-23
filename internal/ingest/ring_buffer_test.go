package ingest

import (
	"sync"
	"testing"
)

func TestRingBuffer_PushPop(t *testing.T) {
	rb := NewRingBuffer(4)
	tick := &GoMarketTick{PriceRaw: 100, VolumeRaw: 10, SymbolID: 1, Side: 1}

	if !rb.Push(tick) {
		t.Fatal("expected push to succeed")
	}

	popped, ok := rb.Pop()
	if !ok {
		t.Fatal("expected pop to succeed")
	}
	if popped.PriceRaw != 100 {
		t.Errorf("expected price 100, got %d", popped.PriceRaw)
	}
	if popped.VolumeRaw != 10 {
		t.Errorf("expected volume 10, got %d", popped.VolumeRaw)
	}
}

func TestRingBuffer_Full(t *testing.T) {
	rb := NewRingBuffer(2)

	rb.Push(&GoMarketTick{PriceRaw: 1})
	rb.Push(&GoMarketTick{PriceRaw: 2})

	if rb.Push(&GoMarketTick{PriceRaw: 3}) {
		t.Error("expected push to fail on full buffer")
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(4)

	_, ok := rb.Pop()
	if ok {
		t.Error("expected pop to fail on empty buffer")
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := NewRingBuffer(4)

	for i := 0; i < 3; i++ {
		rb.Push(&GoMarketTick{PriceRaw: int64(i)})
	}

	for i := 0; i < 3; i++ {
		tick, ok := rb.Pop()
		if !ok {
			t.Fatalf("expected pop to succeed at index %d", i)
		}
		if tick.PriceRaw != int64(i) {
			t.Errorf("expected %d, got %d", i, tick.PriceRaw)
		}
	}

	for i := 0; i < 2; i++ {
		if !rb.Push(&GoMarketTick{PriceRaw: int64(i + 10)}) {
			t.Fatalf("expected push to succeed after wrap at %d", i)
		}
	}

	tick, ok := rb.Pop()
	if !ok || tick.PriceRaw != 10 {
		t.Errorf("expected 10 after wrap, got %d (ok=%v)", tick.PriceRaw, ok)
	}
}

func TestRingBuffer_ConcurrentPushPop(t *testing.T) {
	rb := NewRingBuffer(1024)
	var wg sync.WaitGroup
	n := 1000

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			for !rb.Push(&GoMarketTick{PriceRaw: int64(i)}) {
			}
		}
	}()

	received := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for received < n {
			if _, ok := rb.Pop(); ok {
				received++
			}
		}
	}()

	wg.Wait()
	if received != n {
		t.Errorf("expected %d received, got %d", n, received)
	}
}

func TestRingBuffer_Capacity(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{1, 1}, {2, 2}, {3, 4}, {4, 4},
		{5, 8}, {100, 128}, {1000, 1024},
	}
	for _, tt := range tests {
		rb := NewRingBuffer(tt.input)
		if rb.Capacity() != tt.expected {
			t.Errorf("NewRingBuffer(%d) capacity = %d, want %d", tt.input, rb.Capacity(), tt.expected)
		}
	}
}
