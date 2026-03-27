package engine

import "github.com/imran-binhasan/warpdb/wal"

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