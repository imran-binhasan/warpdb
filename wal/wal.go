package wal

import (
	"encoding/json"
	"io"
	"os"
)

type Entry struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type walEntry struct {
	entry Entry
	done  chan error
}

type WAL struct {
	file    *os.File
	encoder *json.Encoder
	pending chan walEntry
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
	w := &WAL{
		file:    file,
		encoder: json.NewEncoder(file),
		pending: make(chan walEntry, 1024),
	}
	go w.runFlusher()
	return w, nil
}

func (w *WAL) Write(op string, args []string) error {
	e := walEntry{
		entry: Entry{Op: op, Args: args},
		done:  make(chan error, 1),
	}
	w.pending <- e
	return <-e.done
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

func (w *WAL) runFlusher() {
	for {
		first, ok := <-w.pending
		if !ok {
			return
		}

		batch := []walEntry{first}

	drain:
		for {
			select {
			case e, ok := <-w.pending:
				if !ok {
					break drain
				}
				batch = append(batch, e)
			default:
				break drain
			}
		}

		var writeErr error
		for _, e := range batch {
			if err := w.encoder.Encode(e.entry); err != nil {
				writeErr = err
				break
			}
		}

		var syncErr error
		if writeErr == nil {
			syncErr = w.file.Sync()
		}

		for _, e := range batch {
			if writeErr != nil {
				e.done <- writeErr
			} else {
				e.done <- syncErr
			}
		}

	}
}

func (w *WAL) Close() error {
	close(w.pending)
	return w.file.Close()
}
