package engine

import "github.com/imran-binhasan/warpdb/core/store"

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
