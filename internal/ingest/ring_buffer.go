package ingest

import (
	"sync/atomic"
	"unsafe"
)

const MaxTickBuffer = 16384

type GoMarketTick struct {
	Timestamp int64
	PriceRaw  int64
	BidPrice  int64
	AskPrice  int64
	VolumeRaw uint64
	BidSize   uint64
	AskSize   uint64
	SymbolID  uint32
	Side      uint8
	_pad      [3]byte
}

type RingBuffer struct {
	head     int32
	tail     int32
	capacity int32
	_pad     int32
	ticks    [MaxTickBuffer]GoMarketTick
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = MaxTickBuffer
	}
	cap := nextPowerOfTwo(capacity)
	if cap > MaxTickBuffer {
		cap = MaxTickBuffer
	}
	return &RingBuffer{
		capacity: int32(cap),
	}
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func (rb *RingBuffer) Push(tick *GoMarketTick) bool {
	head := atomic.LoadInt32(&rb.head)
	tail := atomic.LoadInt32(&rb.tail)
	next := (tail + 1) % rb.capacity
	if next == head {
		return false
	}
	rb.ticks[tail] = *tick
	atomic.StoreInt32(&rb.tail, next)
	return true
}

func (rb *RingBuffer) Pop() (*GoMarketTick, bool) {
	head := atomic.LoadInt32(&rb.head)
	tail := atomic.LoadInt32(&rb.tail)
	if head == tail {
		return nil, false
	}
	tick := &rb.ticks[head]
	atomic.StoreInt32(&rb.head, (head+1)%rb.capacity)
	return tick, true
}

func (rb *RingBuffer) Pointer() unsafe.Pointer {
	return unsafe.Pointer(rb)
}

func (rb *RingBuffer) Capacity() int {
	return int(rb.capacity)
}

func (rb *RingBuffer) OverflowCount() int32 {
	delta := atomic.LoadInt32(&rb.tail) - atomic.LoadInt32(&rb.head)
	if delta < 0 {
		delta += rb.capacity
	}
	return atomic.LoadInt32(&rb.capacity) - delta
}
