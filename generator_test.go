package sfid

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type mockClock struct {
	mu   sync.Mutex
	cond *sync.Cond
	now  time.Time
}

func newMockClock(now time.Time) *mockClock {
	clk := &mockClock{now: now.UTC()}
	clk.cond = sync.NewCond(&clk.mu)
	return clk
}

func (c *mockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockClock) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}

	c.mu.Lock()
	target := c.now.Add(d)
	for c.now.Before(target) {
		c.cond.Wait()
	}
	c.mu.Unlock()
}

func (c *mockClock) Set(now time.Time) {
	c.mu.Lock()
	c.now = now.UTC()
	c.cond.Broadcast()
	c.mu.Unlock()
}

func (c *mockClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.cond.Broadcast()
	c.mu.Unlock()
}

func mustTestGenerator(t *testing.T, now time.Time, machine uint16, prefix byte) (*Generator, *mockClock) {
	t.Helper()

	clk := newMockClock(now)
	g, err := newGenerator(machine, prefix, clk)
	if err != nil {
		t.Fatalf("newGenerator() error = %v", err)
	}

	return g, clk
}

func TestNewGeneratorValidation(t *testing.T) {
	if _, err := NewGenerator(maxMachine+1, 'd'); !errors.Is(err, ErrInvalidMachine) {
		t.Fatalf("NewGenerator() error = %v, want ErrInvalidMachine", err)
	}

	if _, err := NewGenerator(1, 'D'); !errors.Is(err, ErrInvalidPrefix) {
		t.Fatalf("NewGenerator() error = %v, want ErrInvalidPrefix", err)
	}

	clk := newMockClock(epochTime.Add(-time.Millisecond))
	if _, err := newGenerator(1, 'd', clk); !errors.Is(err, ErrTimeBeforeEpoch) {
		t.Fatalf("newGenerator(before epoch) error = %v, want ErrTimeBeforeEpoch", err)
	}

	clk = newMockClock(timeForBucket(maxTimestamp + 1))
	if _, err := newGenerator(1, 'd', clk); !errors.Is(err, ErrTimeOverflow) {
		t.Fatalf("newGenerator(overflow) error = %v, want ErrTimeOverflow", err)
	}
}

func TestNewGeneratorAcceptsNibblePrefix(t *testing.T) {
	g, _ := mustTestGenerator(t, timeForBucket(10), 7, 0x03)
	id := g.New()

	if got := id.Prefix(); got != 'd' {
		t.Fatalf("Prefix() = %q, want %q", got, 'd')
	}
}

func TestGeneratorSameBucketCounter(t *testing.T) {
	g, _ := mustTestGenerator(t, timeForBucket(100), 42, 'd')

	first := g.New()
	second := g.New()

	if !first.Time().Equal(second.Time()) {
		t.Fatalf("same bucket time mismatch: %v vs %v", first.Time(), second.Time())
	}

	if first.Tick() != 0 || second.Tick() != 0 {
		t.Fatalf("unexpected tick values: %d %d", first.Tick(), second.Tick())
	}

	if first.Counter() != 0 || second.Counter() != 1 {
		t.Fatalf("unexpected counters: %d %d", first.Counter(), second.Counter())
	}
}

func TestGeneratorForwardProgressionResetsCounter(t *testing.T) {
	g, clk := mustTestGenerator(t, timeForBucket(200), 42, 'd')

	first := g.New()
	clk.Advance(bucketDuration)
	second := g.New()

	if second.Counter() != 0 {
		t.Fatalf("Counter() = %d, want 0 after forward bucket change", second.Counter())
	}

	if !second.Time().After(first.Time()) {
		t.Fatalf("Time() = %v, want after %v", second.Time(), first.Time())
	}
}

func TestGeneratorOrderingUnderForwardClockProgression(t *testing.T) {
	g, clk := mustTestGenerator(t, timeForBucket(300), 1, 'a')

	id1 := g.New()
	id2 := g.New()
	clk.Advance(bucketDuration)
	id3 := g.New()

	if int64(id1) >= int64(id2) || int64(id2) >= int64(id3) {
		t.Fatalf("numeric order not preserved: %d, %d, %d", int64(id1), int64(id2), int64(id3))
	}

	if id1.String() >= id2.String() || id2.String() >= id3.String() {
		t.Fatalf("string order not preserved: %q, %q, %q", id1.String(), id2.String(), id3.String())
	}
}

func TestGeneratorRegressionFlipsTickAndPreservesUniqueness(t *testing.T) {
	g, clk := mustTestGenerator(t, timeForBucket(400), 9, 'd')

	first := g.New()
	second := g.New()
	clk.Set(timeForBucket(399))
	third := g.New()

	if first.Tick() != 0 || second.Tick() != 0 {
		t.Fatalf("unexpected initial ticks: %d %d", first.Tick(), second.Tick())
	}

	if third.Tick() != 1 {
		t.Fatalf("Tick() = %d, want 1 after backward regression", third.Tick())
	}

	if !third.Time().Equal(timeForBucket(399)) {
		t.Fatalf("Time() = %v, want %v", third.Time(), timeForBucket(399))
	}

	seen := map[ID]struct{}{
		first:  {},
		second: {},
	}
	if _, ok := seen[third]; ok {
		t.Fatal("third ID collides with a previously generated ID")
	}
}

func TestGeneratorUnsafeRegressionWaitsForFreshTimeline(t *testing.T) {
	g, clk := mustTestGenerator(t, timeForBucket(10), 3, 'b')

	_ = g.New()
	clk.Set(timeForBucket(12))
	idA := g.New()
	if idA.Tick() != 0 {
		t.Fatalf("Tick() = %d, want 0", idA.Tick())
	}

	clk.Set(timeForBucket(8))
	idB := g.New()
	if idB.Tick() != 1 {
		t.Fatalf("Tick() = %d, want 1", idB.Tick())
	}

	clk.Set(timeForBucket(20))
	idC := g.New()
	if idC.Tick() != 1 {
		t.Fatalf("Tick() = %d, want 1", idC.Tick())
	}

	clk.Set(timeForBucket(12))
	result := make(chan ID, 1)
	go func() {
		result <- g.New()
	}()

	select {
	case id := <-result:
		t.Fatalf("generator returned early with %v", id)
	case <-time.After(25 * time.Millisecond):
	}

	clk.Set(timeForBucket(13))

	select {
	case id := <-result:
		if id.Tick() != 0 {
			t.Fatalf("Tick() = %d, want 0 after safe alternate timeline resumes", id.Tick())
		}
		if !id.Time().Equal(timeForBucket(13)) {
			t.Fatalf("Time() = %v, want %v", id.Time(), timeForBucket(13))
		}
	case <-time.After(time.Second):
		t.Fatal("generator did not unblock after fresh bucket became available")
	}
}

func TestGeneratorCounterOverflowWaits(t *testing.T) {
	g, clk := mustTestGenerator(t, timeForBucket(30), 7, 'd')

	var last ID
	for i := 0; i <= int(maxCounter); i++ {
		last = g.New()
	}

	if last.Counter() != maxCounter {
		t.Fatalf("Counter() = %d, want %d", last.Counter(), maxCounter)
	}

	result := make(chan ID, 1)
	go func() {
		result <- g.New()
	}()

	select {
	case id := <-result:
		t.Fatalf("generator returned early with %v", id)
	case <-time.After(25 * time.Millisecond):
	}

	clk.Advance(bucketDuration)

	select {
	case id := <-result:
		if id.Counter() != 0 {
			t.Fatalf("Counter() = %d, want 0 after overflow wait", id.Counter())
		}
		if !id.Time().Equal(timeForBucket(31)) {
			t.Fatalf("Time() = %v, want %v", id.Time(), timeForBucket(31))
		}
	case <-time.After(time.Second):
		t.Fatal("generator did not unblock after counter overflow wait")
	}
}

func TestGeneratorConcurrentUniqueness(t *testing.T) {
	g := MustNewGenerator(17, 'a')

	const goroutines = 32
	const perGoroutine = 4096

	ids := make([]int64, goroutines*perGoroutine)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for n := 0; n < goroutines; n++ {
		offset := n * perGoroutine
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				ids[offset+i] = int64(g.New())
			}
		}(offset)
	}

	wg.Wait()

	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate ID detected: %d", id)
		}
		seen[id] = struct{}{}
	}
}
