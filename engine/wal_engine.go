package engine

import (
	"log/slog"
	"time"

	"github.com/imran-binhasan/warpdb/wal"
)

type WALEngine struct {
	mem *MemoryEngine
	wal *wal.WAL
}

func NewWALEngine(walDir string, maxSizeMB int) (*WALEngine, error) {
	w, err := wal.NewWAL(walDir, maxSizeMB)
	if err != nil {
		return nil, err
	}
	engine := &WALEngine{
		mem: NewMemoryEngine(),
		wal: w,
	}
	if err := w.Replay(engine.mem); err != nil {
		w.Close()
		return nil, err
	}
	slog.Info("WAL replay complete")
	return engine, nil
}

func (e *WALEngine) Set(key, value string) error {
	if err := e.wal.Write("SET", []string{key, value}); err != nil {
		return err
	}
	return e.mem.Set(key, value)
}

func (e *WALEngine) Get(key string) (string, error) {
	return e.mem.Get(key)
}

func (e *WALEngine) Del(key string) error {
	if err := e.wal.Write("DEL", []string{key}); err != nil {
		return err
	}
	return e.mem.Del(key)
}

func (e *WALEngine) Exists(key string) bool {
	return e.mem.Exists(key)
}

func (e *WALEngine) Incr(key string) (int, error) {
	if err := e.wal.Write("INCR", []string{key}); err != nil {
		return 0, err
	}
	return e.mem.Incr(key)
}

func (e *WALEngine) Decr(key string) (int, error) {
	if err := e.wal.Write("DECR", []string{key}); err != nil {
		return 0, err
	}
	return e.mem.Decr(key)
}

func (e *WALEngine) Expire(key string, ttl time.Duration) error {
	return e.mem.Expire(key, ttl)
}

func (e *WALEngine) TTL(key string) (time.Duration, error) {
	return e.mem.TTL(key)
}

func (e *WALEngine) Persist(key string) error {
	return e.mem.Persist(key)
}

func (e *WALEngine) Keys(pattern string) []string {
	return e.mem.Keys(pattern)
}

func (e *WALEngine) FlushAll() {
	if err := e.wal.Write("FLUSHALL", nil); err != nil {
		slog.Error("WAL flushall write failed", "error", err)
	}
	e.mem.FlushAll()
}

func (e *WALEngine) Size() int {
	return e.mem.Size()
}

func (e *WALEngine) LPush(key string, values ...string) int {
	for _, v := range values {
		if err := e.wal.Write("LPUSH", []string{key, v}); err != nil {
			slog.Error("WAL lpush write failed", "error", err)
		}
	}
	return e.mem.LPush(key, values...)
}

func (e *WALEngine) RPush(key string, values ...string) int {
	for _, v := range values {
		if err := e.wal.Write("RPUSH", []string{key, v}); err != nil {
			slog.Error("WAL rpush write failed", "error", err)
		}
	}
	return e.mem.RPush(key, values...)
}

func (e *WALEngine) LPop(key string) (string, error) {
	val, err := e.mem.LPop(key)
	if err != nil {
		return "", err
	}
	if err := e.wal.Write("LPOP", []string{key}); err != nil {
		slog.Error("WAL lpop write failed", "error", err)
	}
	return val, nil
}

func (e *WALEngine) RPop(key string) (string, error) {
	val, err := e.mem.RPop(key)
	if err != nil {
		return "", err
	}
	if err := e.wal.Write("RPOP", []string{key}); err != nil {
		slog.Error("WAL rpop write failed", "error", err)
	}
	return val, nil
}

func (e *WALEngine) LLen(key string) int {
	return e.mem.LLen(key)
}

func (e *WALEngine) LRange(key string, start, stop int) []string {
	return e.mem.LRange(key, start, stop)
}

func (e *WALEngine) LIndex(key string, index int) (string, error) {
	return e.mem.LIndex(key, index)
}

func (e *WALEngine) SAdd(key string, members ...string) int {
	for _, m := range members {
		if err := e.wal.Write("SADD", []string{key, m}); err != nil {
			slog.Error("WAL sadd write failed", "error", err)
		}
	}
	return e.mem.SAdd(key, members...)
}

func (e *WALEngine) SRem(key string, members ...string) int {
	for _, m := range members {
		if err := e.wal.Write("SREM", []string{key, m}); err != nil {
			slog.Error("WAL srem write failed", "error", err)
		}
	}
	return e.mem.SRem(key, members...)
}

func (e *WALEngine) SMembers(key string) []string {
	return e.mem.SMembers(key)
}

func (e *WALEngine) SIsMember(key, member string) bool {
	return e.mem.SIsMember(key, member)
}

func (e *WALEngine) SCard(key string) int {
	return e.mem.SCard(key)
}

func (e *WALEngine) SPop(key string) (string, error) {
	val, err := e.mem.SPop(key)
	if err != nil {
		return "", err
	}
	if err := e.wal.Write("SPOP", []string{key}); err != nil {
		slog.Error("WAL spop write failed", "error", err)
	}
	return val, nil
}

func (e *WALEngine) SRandMember(key string) (string, error) {
	return e.mem.SRandMember(key)
}

func (e *WALEngine) HSet(key, field, value string) int {
	if err := e.wal.Write("HSET", []string{key, field, value}); err != nil {
		slog.Error("WAL hset write failed", "error", err)
	}
	return e.mem.HSet(key, field, value)
}

func (e *WALEngine) HGet(key, field string) (string, error) {
	return e.mem.HGet(key, field)
}

func (e *WALEngine) HDel(key string, fields ...string) int {
	for _, f := range fields {
		if err := e.wal.Write("HDEL", []string{key, f}); err != nil {
			slog.Error("WAL hdel write failed", "error", err)
		}
	}
	return e.mem.HDel(key, fields...)
}

func (e *WALEngine) HGetAll(key string) []string {
	return e.mem.HGetAll(key)
}

func (e *WALEngine) HKeys(key string) []string {
	return e.mem.HKeys(key)
}

func (e *WALEngine) HVals(key string) []string {
	return e.mem.HVals(key)
}

func (e *WALEngine) HExists(key, field string) bool {
	return e.mem.HExists(key, field)
}

func (e *WALEngine) HLen(key string) int {
	return e.mem.HLen(key)
}

func (e *WALEngine) HMSet(key string, fields map[string]string) {
	for f, v := range fields {
		if err := e.wal.Write("HSET", []string{key, f, v}); err != nil {
			slog.Error("WAL hset write failed", "error", err)
		}
	}
	e.mem.HMSet(key, fields)
}

func (e *WALEngine) HMGet(key string, fields ...string) []string {
	return e.mem.HMGet(key, fields...)
}
