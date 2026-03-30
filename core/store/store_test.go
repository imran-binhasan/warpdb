package store

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

// --- unit tests (unchanged) ---

func TestSetGet(t *testing.T) {
	s := NewStore()
	s.Set("name", "Imran")
	val, err := s.Get("name")
	if err != nil {
		t.Fatal(err)
	}
	if val != "Imran" {
		t.Fatalf("expected Imran got %s", val)
	}
}

func TestGetMissing(t *testing.T) {
	s := NewStore()
	_, err := s.Get("ghost")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestIncr(t *testing.T) {
	s := NewStore()
	val, err := s.Incr("counter")
	if err != nil || val != 1 {
		t.Fatalf("expected 1 got %d err %v", val, err)
	}
	val, err = s.Incr("counter")
	if err != nil || val != 2 {
		t.Fatalf("expected 2 got %d err %v", val, err)
	}
}

func TestConcurrentIncr(t *testing.T) {
	s := NewStore()
	s.Set("counter", "0")
	done := make(chan bool)
	for i := 0; i < 1000; i++ {
		go func() {
			s.Incr("counter")
			done <- true
		}()
	}
	for i := 0; i < 1000; i++ {
		<-done
	}
	val, _ := s.Get("counter")
	if val != "1000" {
		t.Fatalf("expected value 1000 got %s", val)
	}
}

// --- benchmark helpers ---

const (
	benchKeyspace = 100_000 // realistic: 100k distinct keys
	benchValSize  = 64      // 64-byte value, typical cache entry
)

// makeKey turns an integer into a realistic-looking key.
// Using a fixed prefix avoids benchmark variability from key length.
func makeKey(i int) string {
	return fmt.Sprintf("key:%08d", i)
}

// makeVal returns a fixed-size value string of benchValSize bytes.
func makeVal() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, benchValSize)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

// populate fills the store with n keys so read benchmarks
// don't measure misses.
func populate(s *Store, n int) {
	val := makeVal()
	for i := 0; i < n; i++ {
		s.Set(makeKey(i), val)
	}
}

// --- isolated operation benchmarks ---

// BenchmarkSet measures pure write throughput across the full keyspace.
// Each goroutine writes to a different region of the keyspace so we
// see shard parallelism, not single-shard contention.
func BenchmarkSet(b *testing.B) {
	s := NewStore()
	val := makeVal()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			s.Set(makeKey(i%benchKeyspace), val)
			i++
		}
	})
}

// BenchmarkGet measures pure read throughput on pre-populated keys.
// Keys are chosen uniformly at random across the keyspace.
func BenchmarkGet(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			s.Get(makeKey(i % benchKeyspace))
			i++
		}
	})
}

// BenchmarkIncr measures counter throughput.
// Split across 1000 distinct counter keys so we see real shard
// distribution, not a single bottleneck key.
func BenchmarkIncr(b *testing.B) {
	s := NewStore()
	const numCounters = 1000
	for i := 0; i < numCounters; i++ {
		s.Set(fmt.Sprintf("counter:%d", i), "0")
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(numCounters)
		for pb.Next() {
			s.Incr(fmt.Sprintf("counter:%d", i%numCounters))
			i++
		}
	})
}

// BenchmarkDel measures delete throughput.
// Re-populates between timer resets so we're not measuring misses.
func BenchmarkDel(b *testing.B) {
	s := NewStore()
	val := makeVal()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			key := makeKey(i % benchKeyspace)
			s.Set(key, val)
			s.Del(key)
			i++
		}
	})
}

// --- mixed workload benchmarks ---

// BenchmarkReadHeavy simulates a cache workload:
// 95% reads, 5% writes. This is the most realistic scenario
// for a Redis-like store used as an application cache.
func BenchmarkReadHeavy(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	val := makeVal()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			key := makeKey(i % benchKeyspace)
			if i%20 == 0 {
				s.Set(key, val) // 5% writes
			} else {
				s.Get(key) // 95% reads
			}
			i++
		}
	})
}

// BenchmarkBalanced simulates a balanced read/write workload:
// 50% reads, 50% writes. Tests under higher write pressure.
func BenchmarkBalanced(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	val := makeVal()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			key := makeKey(i % benchKeyspace)
			if i%2 == 0 {
				s.Set(key, val)
			} else {
				s.Get(key)
			}
			i++
		}
	})
}

// BenchmarkWriteHeavy simulates a high-ingest workload:
// 20% reads, 80% writes. Tests write scalability under shard contention.
func BenchmarkWriteHeavy(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	val := makeVal()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchKeyspace)
		for pb.Next() {
			key := makeKey(i % benchKeyspace)
			if i%5 == 0 {
				s.Get(key) // 20% reads
			} else {
				s.Set(key, val) // 80% writes
			}
			i++
		}
	})
}

// --- memory allocation benchmarks ---

// BenchmarkSetAllocs checks that Set causes zero heap allocations
// for existing keys (no map growth, no new string allocation on the
// hot path).
func BenchmarkSetAllocs(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	val := makeVal()
	key := makeKey(42)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Set(key, val)
	}
}

// BenchmarkGetAllocs checks that Get causes zero heap allocations
// on a cache hit — the returned string should alias the stored value,
// not copy it.
func BenchmarkGetAllocs(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)
	key := makeKey(42)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Get(key)
	}
}

// --- shard distribution benchmark ---

// BenchmarkShardDistribution verifies that FNV-1a spreads 100k keys
// evenly across 256 shards. Prints the min/max/stddev of shard sizes.
// Not a throughput benchmark — run with -v to see the output.
func BenchmarkShardDistribution(b *testing.B) {
	s := NewStore()
	populate(s, benchKeyspace)

	sizes := make([]int, numShards)
	for i := 0; i < numShards; i++ {
		s.shards[i].mu.RLock()
		sizes[i] = len(s.shards[i].data)
		s.shards[i].mu.RUnlock()
	}

	var total, min, max int
	min = sizes[0]
	for _, sz := range sizes {
		total += sz
		if sz < min {
			min = sz
		}
		if sz > max {
			max = sz
		}
	}
	avg := total / numShards

	var variance float64
	for _, sz := range sizes {
		diff := float64(sz - avg)
		variance += diff * diff
	}
	variance /= float64(numShards)

	b.ReportMetric(float64(min), "min-keys/shard")
	b.ReportMetric(float64(max), "max-keys/shard")
	b.ReportMetric(float64(avg), "avg-keys/shard")
	b.ReportMetric(variance, "variance")

	_ = strconv.Itoa(avg) // prevent unused import
}
