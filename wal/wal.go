package wal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type WAL struct {
	dir       string
	file      *os.File
	pending   chan walEntry
	seq       int
	maxSize   int64
	currentSz int64
	encoder   *RecordEncoder
}

type walEntry struct {
	record Record
	done   chan error
}

type Replayer interface {
	Set(key, value string) error
	Del(key string) error
	Incr(key string) (int, error)
	Decr(key string) (int, error)
	FlushAll()
}

type segmentManifest struct {
	Segments []string `json:"segments"`
}

func NewWAL(dir string, maxSizeMB int) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	manifest, err := readManifest(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	w := &WAL{
		dir:     dir,
		pending: make(chan walEntry, 1024),
		maxSize: int64(maxSizeMB) * 1024 * 1024,
	}

	if len(manifest.Segments) > 0 {
		lastSeg := manifest.Segments[len(manifest.Segments)-1]
		seqStr := strings.TrimSuffix(filepath.Base(lastSeg), ".wal")
		w.seq, _ = strconv.Atoi(seqStr)

		f, err := os.OpenFile(filepath.Join(dir, lastSeg), os.O_APPEND|os.O_RDWR, 0644)
		if err != nil {
			return nil, err
		}
		w.file = f
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}
		w.currentSz = fi.Size()
	} else {
		if err := w.rotate(); err != nil {
			return nil, err
		}
	}

	w.encoder = NewRecordEncoder(w.file)
	go w.runFlusher()
	return w, nil
}

func (w *WAL) rotate() error {
	if w.file != nil {
		w.file.Close()
	}

	w.seq++
	segName := fmt.Sprintf("%08d.wal", w.seq)
	path := filepath.Join(w.dir, segName)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	w.file = f
	w.currentSz = 0
	w.encoder = NewRecordEncoder(f)

	return w.writeManifest()
}

func (w *WAL) Write(op string, args []string) error {
	opCode := opToCode(op)
	r := MakeRecord(opCode, args)
	e := walEntry{
		record: r,
		done:   make(chan error, 1),
	}
	w.pending <- e
	return <-e.done
}

func (w *WAL) Sync() error {
	return w.file.Sync()
}

func (w *WAL) Replay(engine Replayer) error {
	manifest, err := readManifest(w.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, seg := range manifest.Segments {
		if err := w.replaySegment(engine, seg); err != nil {
			return fmt.Errorf("replay segment %s: %w", seg, err)
		}
	}
	return nil
}

func (w *WAL) replaySegment(engine Replayer, seg string) error {
	f, err := os.Open(filepath.Join(w.dir, seg))
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := NewRecordDecoder(f)
	for {
		record, err := decoder.Decode()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}

		switch record.Op {
		case OpSet:
			engine.Set(record.Key, record.Value)
		case OpDel:
			engine.Del(record.Key)
		case OpIncr:
			engine.Incr(record.Key)
		case OpDecr:
			engine.Decr(record.Key)
		case OpFlush:
			engine.FlushAll()
		}
	}
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
		totalWritten := int64(0)
		for _, e := range batch {
			n, err := w.encoder.Encode(e.record)
			if err != nil {
				writeErr = err
				break
			}
			totalWritten += int64(n)
		}

		var syncErr error
		if writeErr == nil {
			syncErr = w.file.Sync()
		}

		w.currentSz += totalWritten

		if w.maxSize > 0 && w.currentSz >= w.maxSize {
			if err := w.rotate(); err != nil {
				if writeErr == nil {
					writeErr = err
				}
			}
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
	if w.file != nil {
		w.file.Close()
	}
	return w.writeManifest()
}

func (w *WAL) Compact() error {
	manifest, err := readManifest(w.dir)
	if err != nil {
		return err
	}
	if len(manifest.Segments) <= 1 {
		return nil
	}

	latest := make(map[string]Record)
	for _, seg := range manifest.Segments {
		f, err := os.Open(filepath.Join(w.dir, seg))
		if err != nil {
			return err
		}
		decoder := NewRecordDecoder(f)
		for {
			record, err := decoder.Decode()
			if err == io.EOF {
				break
			}
			if err != nil {
				f.Close()
				return err
			}
			if record.Op == OpDel || record.Op == OpFlush {
				delete(latest, record.Key)
			} else {
				latest[record.Key] = record
			}
		}
		f.Close()
	}

	compactedSeg := fmt.Sprintf("%08d.wal", w.seq+1)
	path := filepath.Join(w.dir, compactedSeg)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	enc := NewRecordEncoder(f)
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := enc.Encode(latest[k]); err != nil {
			f.Close()
			return err
		}
	}
	f.Sync()
	f.Close()

	for _, seg := range manifest.Segments {
		os.Remove(filepath.Join(w.dir, seg))
	}

	newManifest := segmentManifest{Segments: []string{compactedSeg}}
	return writeManifestFile(w.dir, newManifest)
}

func (w *WAL) writeManifest() error {
	manifest, err := readManifest(w.dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	currentSeg := fmt.Sprintf("%08d.wal", w.seq)
	found := false
	for _, seg := range manifest.Segments {
		if seg == currentSeg {
			found = true
			break
		}
	}
	if !found {
		manifest.Segments = append(manifest.Segments, currentSeg)
	}

	return writeManifestFile(w.dir, manifest)
}

func readManifest(dir string) (segmentManifest, error) {
	var m segmentManifest
	f, err := os.Open(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return m, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&m)
	return m, err
}

func writeManifestFile(dir string, m segmentManifest) error {
	f, err := os.OpenFile(filepath.Join(dir, "manifest.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

func opToCode(op string) byte {
	switch op {
	case "SET":
		return OpSet
	case "DEL":
		return OpDel
	case "INCR":
		return OpIncr
	case "DECR":
		return OpDecr
	case "FLUSHALL":
		return OpFlush
	default:
		return OpSet
	}
}
