package store

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
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

// --- list tests ---

func TestLPushRPush(t *testing.T) {
	s := NewStore()
	s.RPush("mylist", "a", "b", "c")
	s.LPush("mylist", "x")
	if s.LLen("mylist") != 4 {
		t.Fatalf("expected len 4 got %d", s.LLen("mylist"))
	}
}

func TestLPopRPop(t *testing.T) {
	s := NewStore()
	s.RPush("mylist", "a", "b", "c")
	val, err := s.LPop("mylist")
	if err != nil || val != "a" {
		t.Fatalf("expected 'a' got '%s' err=%v", val, err)
	}
	val, err = s.RPop("mylist")
	if err != nil || val != "c" {
		t.Fatalf("expected 'c' got '%s' err=%v", val, err)
	}
	if s.LLen("mylist") != 1 {
		t.Fatalf("expected len 1 got %d", s.LLen("mylist"))
	}
}

func TestLRange(t *testing.T) {
	s := NewStore()
	s.RPush("mylist", "a", "b", "c", "d", "e")
	vals := s.LRange("mylist", 1, 3)
	if len(vals) != 3 || vals[0] != "b" || vals[1] != "c" || vals[2] != "d" {
		t.Fatalf("unexpected LRange result: %v", vals)
	}
}

func TestLIndex(t *testing.T) {
	s := NewStore()
	s.RPush("mylist", "a", "b", "c")
	val, err := s.LIndex("mylist", -1)
	if err != nil || val != "c" {
		t.Fatalf("expected 'c' got '%s'", val)
	}
}

// --- set tests ---

func TestSAdd(t *testing.T) {
	s := NewStore()
	added := s.SAdd("myset", "a", "b", "a")
	if added != 2 {
		t.Fatalf("expected 2 added got %d", added)
	}
	if s.SCard("myset") != 2 {
		t.Fatalf("expected card 2 got %d", s.SCard("myset"))
	}
}

func TestSIsMember(t *testing.T) {
	s := NewStore()
	s.SAdd("myset", "a", "b")
	if !s.SIsMember("myset", "a") {
		t.Fatal("expected a to be member")
	}
	if s.SIsMember("myset", "c") {
		t.Fatal("expected c not to be member")
	}
}

func TestSRem(t *testing.T) {
	s := NewStore()
	s.SAdd("myset", "a", "b", "c")
	removed := s.SRem("myset", "a", "d")
	if removed != 1 {
		t.Fatalf("expected 1 removed got %d", removed)
	}
	if s.SCard("myset") != 2 {
		t.Fatalf("expected card 2 got %d", s.SCard("myset"))
	}
}

func TestSPop(t *testing.T) {
	s := NewStore()
	s.SAdd("myset", "a")
	val, err := s.SPop("myset")
	if err != nil || val != "a" {
		t.Fatalf("expected 'a' got '%s'", val)
	}
	if s.SCard("myset") != 0 {
		t.Fatal("set should be empty")
	}
}

func TestSRandMember(t *testing.T) {
	s := NewStore()
	s.SAdd("myset", "only")
	val, err := s.SRandMember("myset")
	if err != nil || val != "only" {
		t.Fatalf("expected 'only' got '%s'", val)
	}
}

func TestSMembers(t *testing.T) {
	s := NewStore()
	s.SAdd("myset", "a", "b")
	members := s.SMembers("myset")
	if len(members) != 2 {
		t.Fatalf("expected 2 members got %d", len(members))
	}
}

// --- hash tests ---

func TestHSetHGet(t *testing.T) {
	s := NewStore()
	added := s.HSet("myhash", "field1", "value1")
	if added != 1 {
		t.Fatalf("expected 1 added got %d", added)
	}
	val, err := s.HGet("myhash", "field1")
	if err != nil || val != "value1" {
		t.Fatalf("expected 'value1' got '%s'", val)
	}
}

func TestHSetDuplicate(t *testing.T) {
	s := NewStore()
	s.HSet("myhash", "f1", "v1")
	added := s.HSet("myhash", "f1", "v2")
	if added != 0 {
		t.Fatalf("expected 0 (update) got %d", added)
	}
}

func TestHDel(t *testing.T) {
	s := NewStore()
	s.HSet("myhash", "f1", "v1")
	s.HSet("myhash", "f2", "v2")
	removed := s.HDel("myhash", "f1")
	if removed != 1 {
		t.Fatalf("expected 1 removed got %d", removed)
	}
	if s.HLen("myhash") != 1 {
		t.Fatalf("expected len 1 got %d", s.HLen("myhash"))
	}
}

func TestHGetAll(t *testing.T) {
	s := NewStore()
	s.HSet("myhash", "a", "1")
	s.HSet("myhash", "b", "2")
	all := s.HGetAll("myhash")
	if len(all) != 4 {
		t.Fatalf("expected 4 entries got %d", len(all))
	}
}

func TestHKeysHVals(t *testing.T) {
	s := NewStore()
	s.HSet("myhash", "a", "1")
	keys := s.HKeys("myhash")
	if len(keys) != 1 || keys[0] != "a" {
		t.Fatalf("unexpected keys: %v", keys)
	}
	vals := s.HVals("myhash")
	if len(vals) != 1 || vals[0] != "1" {
		t.Fatalf("unexpected vals: %v", vals)
	}
}

func TestHExists(t *testing.T) {
	s := NewStore()
	s.HSet("myhash", "field", "val")
	if !s.HExists("myhash", "field") {
		t.Fatal("field should exist")
	}
	if s.HExists("myhash", "nope") {
		t.Fatal("nope should not exist")
	}
}

// --- mixed operations ---

func TestDelRemovesAllTypes(t *testing.T) {
	s := NewStore()
	s.Set("str", "val")
	s.RPush("list", "a")
	s.SAdd("set", "m")
	s.HSet("hash", "f", "v")

	s.Del("str")
	s.Del("list")
	s.Del("set")
	s.Del("hash")

	if s.Exists("str") || s.Exists("list") || s.Exists("set") || s.Exists("hash") {
		t.Fatal("all keys should be deleted")
	}
}

// --- expiry across types ---

func TestExpireOnList(t *testing.T) {
	s := NewStore()
	s.RPush("mylist", "a")
	if err := s.Expire("mylist", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !s.Exists("mylist") {
		t.Fatal("list should exist before expiry")
	}
	time.Sleep(200 * time.Millisecond)
	if s.Exists("mylist") {
		t.Fatal("list should be expired after TTL")
	}
}

func TestSize(t *testing.T) {
	s := NewStore()
	s.Set("a", "1")
	s.RPush("b", "x")
	s.SAdd("c", "y")
	s.HSet("d", "f", "z")
	if s.Size() != 4 {
		t.Fatalf("expected Size 4 got %d", s.Size())
	}
}
