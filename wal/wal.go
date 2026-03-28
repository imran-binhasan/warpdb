package wal

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

type Entry struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type WAL struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

type Replayer interface {
	Set(key, value string) error
	Del(key string) error
	Incr(key string) (int, error)
	Decr(key string) (int, error)
}

func NewWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	encoder := json.NewEncoder(file)
	return &WAL{
		file:    file,
		encoder: encoder,
	}, nil
}

func (w *WAL) Write(op string, args []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	entry := Entry{op, args}
	return w.encoder.Encode(entry)
}

func (w *WAL) Sync() error {
	return w.file.Sync()
}

func (w *WAL) Replay(engine Replayer) error {
	_, err := w.file.Seek(0, 0)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(w.file)
	for {
		var entry Entry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch entry.Op {
		case "SET":
			if len(entry.Args) != 2 {
				continue
			}
			engine.Set(entry.Args[0], entry.Args[1])
		case "DEL":
			if len(entry.Args) != 1 {
				continue
			}
			engine.Del(entry.Args[0])
		case "INCR":
			if len(entry.Args) != 1 {
				continue
			}
			engine.Incr(entry.Args[0])
		case "DECR":
			if len(entry.Args) != 1 {
				continue
			}
			engine.Decr(entry.Args[0])
		}
	}
	return nil
}
