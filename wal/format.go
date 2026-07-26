package wal

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

const (
	entryHeaderSize = 4 + 1 + 2 + 4 + 8 // CRC(4) + Op(1) + KeyLen(2) + ValLen(4) + ExpireAt(8) = 19

	OpSet   byte = 0x01
	OpDel   byte = 0x02
	OpIncr  byte = 0x03
	OpDecr  byte = 0x04
	OpFlush byte = 0x05
)

var crc32Table = crc32.MakeTable(crc32.IEEE)

var ErrChecksumMismatch = errors.New("wal: checksum mismatch")
var ErrShortEntry = errors.New("wal: entry too short")

type Record struct {
	Op       byte
	Key      string
	Value    string
	ExpireAt int64
}

func EncodeRecord(r Record) []byte {
	keyBytes := []byte(r.Key)
	valBytes := []byte(r.Value)

	totalLen := entryHeaderSize + len(keyBytes) + len(valBytes)
	buf := make([]byte, totalLen)

	offset := 4

	buf[offset] = r.Op
	offset++

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(keyBytes)))
	offset += 2

	copy(buf[offset:], keyBytes)
	offset += len(keyBytes)

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(valBytes)))
	offset += 4

	copy(buf[offset:], valBytes)
	offset += len(valBytes)

	binary.BigEndian.PutUint64(buf[offset:], uint64(r.ExpireAt))
	offset += 8

	crc := crc32.Checksum(buf[4:], crc32Table)
	binary.BigEndian.PutUint32(buf[0:4], crc)

	return buf
}

func DecodeRecord(data []byte) (Record, error) {
	if len(data) < entryHeaderSize {
		return Record{}, ErrShortEntry
	}

	storedCRC := binary.BigEndian.Uint32(data[0:4])
	computedCRC := crc32.Checksum(data[4:], crc32Table)
	if storedCRC != computedCRC {
		return Record{}, ErrChecksumMismatch
	}

	offset := 4

	op := data[offset]
	offset++

	keyLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	if offset+keyLen > len(data) {
		return Record{}, ErrShortEntry
	}
	key := string(data[offset : offset+keyLen])
	offset += keyLen

	if offset+4 > len(data) {
		return Record{}, ErrShortEntry
	}
	valLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	if offset+valLen > len(data) {
		return Record{}, ErrShortEntry
	}
	value := string(data[offset : offset+valLen])
	offset += valLen

	if offset+8 > len(data) {
		return Record{}, ErrShortEntry
	}
	expireAt := int64(binary.BigEndian.Uint64(data[offset:]))

	return Record{
		Op:       op,
		Key:      key,
		Value:    value,
		ExpireAt: expireAt,
	}, nil
}

func MakeRecord(op byte, args []string) Record {
	r := Record{Op: op}
	switch op {
	case OpSet:
		if len(args) >= 2 {
			r.Key = args[0]
			r.Value = args[1]
		}
	case OpDel, OpIncr, OpDecr:
		if len(args) >= 1 {
			r.Key = args[0]
		}
	case OpFlush:
	}
	return r
}

func (r Record) Args() []string {
	switch r.Op {
	case OpSet:
		return []string{r.Key, r.Value}
	case OpDel, OpIncr, OpDecr:
		return []string{r.Key}
	case OpFlush:
		return nil
	}
	return nil
}

type RecordEncoder struct {
	w io.Writer
}

func NewRecordEncoder(w io.Writer) *RecordEncoder {
	return &RecordEncoder{w: w}
}

func (e *RecordEncoder) Encode(r Record) (int, error) {
	data := EncodeRecord(r)
	return e.w.Write(data)
}

type RecordDecoder struct {
	r   io.Reader
	buf []byte
}

func NewRecordDecoder(r io.Reader) *RecordDecoder {
	return &RecordDecoder{r: r, buf: make([]byte, 0, 4096)}
}

func (d *RecordDecoder) Decode() (Record, error) {
	header := make([]byte, entryHeaderSize)
	_, err := io.ReadFull(d.r, header)
	if err != nil {
		return Record{}, err
	}

	keyLen := int(binary.BigEndian.Uint16(header[5:7]))
	valLen := int(binary.BigEndian.Uint32(header[7+keyLen : 11+keyLen]))

	bodyLen := keyLen + valLen
	totalLen := entryHeaderSize + bodyLen

	full := make([]byte, totalLen)
	copy(full, header)
	_, err = io.ReadFull(d.r, full[entryHeaderSize:])
	if err != nil {
		return Record{}, err
	}

	return DecodeRecord(full)
}
