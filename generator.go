package sfid

import (
	"fmt"
	"sync"
	"time"
)

type clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Generator generates sfid IDs for a fixed machine and binary prefix.
//
// A Generator is safe for concurrent use by multiple goroutines and must not be
// copied after first use.
type Generator struct {
	mu sync.Mutex

	machine uint16
	prefix  uint8
	clock   clock

	lastBucket int64
	lastTick   uint8
	counter    uint16

	usedThrough [2]int64
}

// NewGenerator constructs a generator for the given machine and binary prefix.
//
// The machine must be in the range 0..1023. The prefix may be either a raw
// nibble in the range 0..7 or the visible lowercase character a-h.
func NewGenerator(machine uint16, prefix byte) (*Generator, error) {
	return newGenerator(machine, prefix, realClock{})
}

// MustNewGenerator constructs a generator or panics if the configuration is invalid.
func MustNewGenerator(machine uint16, prefix byte) *Generator {
	g, err := NewGenerator(machine, prefix)
	if err != nil {
		panic(err)
	}

	return g
}

func newGenerator(machine uint16, prefix byte, clk clock) (*Generator, error) {
	if machine > maxMachine {
		return nil, fmt.Errorf("%w: got %d, want 0..%d", ErrInvalidMachine, machine, maxMachine)
	}

	if clk == nil {
		clk = realClock{}
	}

	prefixNibble, err := normalizePrefix(prefix)
	if err != nil {
		return nil, err
	}

	if _, err := bucketFromTime(clk.Now()); err != nil {
		return nil, err
	}

	return &Generator{
		machine:     machine,
		prefix:      prefixNibble,
		clock:       clk,
		lastBucket:  -1,
		usedThrough: [2]int64{-1, -1},
	}, nil
}

// New generates a new ID.
func (g *Generator) New() ID {
	for {
		id, wait, ok := g.tryNew()
		if ok {
			return id
		}

		g.clock.Sleep(wait)
	}
}

// NewString generates a new ID and returns its canonical string form.
func (g *Generator) NewString() string {
	return g.New().String()
}

func (g *Generator) tryNew() (ID, time.Duration, bool) {
	now := g.clock.Now()

	g.mu.Lock()

	bucket, err := bucketFromTime(now)
	if err != nil {
		wait := g.waitForClockError(now, err)
		g.mu.Unlock()
		return 0, wait, false
	}

	if g.lastBucket < 0 {
		id := g.emitLocked(bucket, 0, 0)
		g.mu.Unlock()
		return id, 0, true
	}

	switch {
	case bucket > g.lastBucket:
		id := g.emitLocked(bucket, g.lastTick, 0)
		g.mu.Unlock()
		return id, 0, true
	case bucket == g.lastBucket:
		if g.counter < maxCounter {
			id := g.emitLocked(bucket, g.lastTick, g.counter+1)
			g.mu.Unlock()
			return id, 0, true
		}

		wait := g.waitUntilBucket(now, g.lastBucket+1)
		g.mu.Unlock()
		return 0, wait, false
	default:
		newTick := g.lastTick ^ 1
		if bucket <= g.usedThrough[newTick] {
			wait := g.waitUntilBucket(now, g.usedThrough[newTick]+1)
			g.mu.Unlock()
			return 0, wait, false
		}

		id := g.emitLocked(bucket, newTick, 0)
		g.mu.Unlock()
		return id, 0, true
	}
}

func (g *Generator) emitLocked(bucket int64, tick uint8, counter uint16) ID {
	g.lastBucket = bucket
	g.lastTick = tick
	g.counter = counter

	if bucket > g.usedThrough[tick] {
		g.usedThrough[tick] = bucket
	}

	return packID(g.prefix, bucket, tick, g.machine, counter)
}

func (g *Generator) waitUntilBucket(now time.Time, bucket int64) time.Duration {
	wait := durationUntilBucket(now, bucket)
	if wait > 0 {
		return wait
	}

	return 0
}

func (g *Generator) waitForClockError(now time.Time, err error) time.Duration {
	switch err {
	case ErrTimeBeforeEpoch:
		wait := epochTime.Sub(now)
		if wait > 0 {
			return wait
		}
	case ErrTimeOverflow:
		return bucketDuration
	}

	return bucketDuration
}
