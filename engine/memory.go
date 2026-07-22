package engine

import (
	"time"

	"github.com/imran-binhasan/warpdb/core/store"
)

type MemoryEngine struct {
	store *store.Store
}

var _ StorageEngine = (*MemoryEngine)(nil)

func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{store: store.NewStore()}
}

func (e *MemoryEngine) Set(key, value string) error {
	e.store.Set(key, value)
	return nil
}

func (e *MemoryEngine) Get(key string) (string, error) {
	return e.store.Get(key)
}

func (e *MemoryEngine) Del(key string) error {
	return e.store.Del(key)
}

func (e *MemoryEngine) Exists(key string) bool {
	return e.store.Exists(key)
}

func (e *MemoryEngine) Incr(key string) (int, error) {
	return e.store.Incr(key)
}

func (e *MemoryEngine) Decr(key string) (int, error) {
	return e.store.Decr(key)
}

func (e *MemoryEngine) Expire(key string, ttl time.Duration) error {
	return e.store.Expire(key, ttl)
}

func (e *MemoryEngine) TTL(key string) (time.Duration, error) {
	return e.store.TTL(key)
}

func (e *MemoryEngine) Persist(key string) error {
	return e.store.Persist(key)
}

func (e *MemoryEngine) Keys(pattern string) []string {
	return e.store.Keys(pattern)
}

func (e *MemoryEngine) FlushAll() {
	e.store.FlushAll()
}

func (e *MemoryEngine) Size() int {
	return e.store.Size()
}
