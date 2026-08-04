package wal

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeSet(t *testing.T) {
	orig := Record{Op: OpSet, Key: "hello", Value: "world"}
	data := EncodeRecord(orig)

	dec, err := DecodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Op != OpSet {
		t.Fatalf("expected OpSet got %d", dec.Op)
	}
	if dec.Key != "hello" {
		t.Fatalf("expected key 'hello' got '%s'", dec.Key)
	}
	if dec.Value != "world" {
		t.Fatalf("expected value 'world' got '%s'", dec.Value)
	}
}

func TestEncodeDecodeDelete(t *testing.T) {
	orig := Record{Op: OpDel, Key: "tempkey"}
	data := EncodeRecord(orig)

	dec, err := DecodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Op != OpDel {
		t.Fatalf("expected OpDel got %d", dec.Op)
	}
	if dec.Key != "tempkey" {
		t.Fatalf("expected key 'tempkey' got '%s'", dec.Key)
	}
}

func TestEncodeDecodeEmptyValue(t *testing.T) {
	orig := Record{Op: OpSet, Key: "empty", Value: ""}
	data := EncodeRecord(orig)

	dec, err := DecodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Value != "" {
		t.Fatalf("expected empty value got '%s'", dec.Value)
	}
}

func TestCRC32Integrity(t *testing.T) {
	orig := Record{Op: OpSet, Key: "data", Value: "important"}
	data := EncodeRecord(orig)

	data[0] ^= 0xFF

	_, err := DecodeRecord(data)
	if err != ErrChecksumMismatch {
		t.Fatalf("expected ErrChecksumMismatch got %v", err)
	}
}

func TestShortEntry(t *testing.T) {
	_, err := DecodeRecord([]byte{1, 2, 3})
	if err != ErrShortEntry {
		t.Fatalf("expected ErrShortEntry got %v", err)
	}
}

func TestRecordEncoderDecoder(t *testing.T) {
	var buf bytes.Buffer
	enc := NewRecordEncoder(&buf)

	records := []Record{
		{Op: OpSet, Key: "k1", Value: "v1"},
		{Op: OpSet, Key: "k2", Value: "v2"},
		{Op: OpDel, Key: "k1"},
	}

	for _, r := range records {
		if _, err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}

	dec := NewRecordDecoder(bytes.NewReader(buf.Bytes()))
	for i, expected := range records {
		got, err := dec.Decode()
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		if got.Op != expected.Op || got.Key != expected.Key || got.Value != expected.Value {
			t.Fatalf("record %d mismatch: got {%d %s %s} expected {%d %s %s}",
				i, got.Op, got.Key, got.Value, expected.Op, expected.Key, expected.Value)
		}
	}
}

func TestMakeRecordSet(t *testing.T) {
	r := MakeRecord(OpSet, []string{"mykey", "myval"})
	if r.Op != OpSet || r.Key != "mykey" || r.Value != "myval" {
		t.Fatalf("unexpected record: %+v", r)
	}
}

func TestMakeRecordHSet(t *testing.T) {
	r := MakeRecord(OpHSet, []string{"hashkey", "field1", "val1"})
	if r.Op != OpHSet || r.Key != "hashkey" {
		t.Fatalf("unexpected record: %+v", r)
	}
	field, val := r.HashFieldAndValue()
	if field != "field1" || val != "val1" {
		t.Fatalf("expected field1=val1 got %s=%s", field, val)
	}
}

func TestMakeRecordFlush(t *testing.T) {
	r := MakeRecord(OpFlush, nil)
	if r.Op != OpFlush {
		t.Fatalf("expected OpFlush got %d", r.Op)
	}
}
