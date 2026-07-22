package engine

import (
	"time"

	"github.com/imran-binhasan/warpdb/wal"
)

type WALEngine struct {
	mem *MemoryEngine
	wal *wal.WAL
}

func NewWALEngine(walPath string) (*WALEngine, error) {
	w, err := wal.NewWAL(walPath)
	if err != nil {
		return nil, err
	}
	engine := &WALEngine{
		mem: NewMemoryEngine(),
		wal: w,
	}
	err = w.Replay(engine.mem)
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *WALEngine) Set(key, value string) error {
	err := e.wal.Write("SET", []string{key, value})
	if err != nil {
		return err
	}
	return e.mem.Set(key, value)
}

func (e *WALEngine) Get(key string) (string, error) {
	return e.mem.Get(key)
}

func (e *WALEngine) Del(key string) error {
	err := e.wal.Write("DEL", []string{key})
	if err != nil {
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
	e.mem.FlushAll()
}

func (e *WALEngine) Size() int {
	return e.mem.Size()
}
