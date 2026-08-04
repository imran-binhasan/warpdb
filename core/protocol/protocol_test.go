package protocol

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestParseSimple(t *testing.T) {
	raw := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	reader := bufio.NewReader(strings.NewReader(raw))
	args, err := Parse(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args got %d", len(args))
	}
	if args[0] != "SET" || args[1] != "key" || args[2] != "value" {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestParsePing(t *testing.T) {
	raw := "*1\r\n$4\r\nPING\r\n"
	reader := bufio.NewReader(strings.NewReader(raw))
	args, err := Parse(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 || args[0] != "PING" {
		t.Fatalf("expected PING got %v", args)
	}
}

func TestParseEmpty(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := Parse(reader)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseInlineCommand(t *testing.T) {
	raw := "PING\r\n"
	reader := bufio.NewReader(strings.NewReader(raw))
	_, err := Parse(reader)
	if err == nil {
		t.Fatal("expected error for inline command")
	}
}

func TestWriteSimpleString(t *testing.T) {
	var buf bytes.Buffer
	WriteSimpleString(&buf, "OK")
	if buf.String() != "+OK\r\n" {
		t.Fatalf("expected '+OK\\r\\n' got %q", buf.String())
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, "ERR message")
	if buf.String() != "-ERR message\r\n" {
		t.Fatalf("expected '-ERR message\\r\\n' got %q", buf.String())
	}
}

func TestWriteInteger(t *testing.T) {
	var buf bytes.Buffer
	WriteInteger(&buf, 42)
	if buf.String() != ":42\r\n" {
		t.Fatalf("expected ':42\\r\\n' got %q", buf.String())
	}
}

func TestWriteBulkString(t *testing.T) {
	var buf bytes.Buffer
	WriteBulkString(&buf, "hello")
	expected := "$5\r\nhello\r\n"
	if buf.String() != expected {
		t.Fatalf("expected %q got %q", expected, buf.String())
	}
}

func TestWriteNull(t *testing.T) {
	var buf bytes.Buffer
	WriteNull(&buf)
	if buf.String() != "$-1\r\n" {
		t.Fatalf("expected '$-1\\r\\n' got %q", buf.String())
	}
}

func TestWriteArray(t *testing.T) {
	var buf bytes.Buffer
	WriteArray(&buf, []string{"a", "bb", "ccc"})
	expected := "*3\r\n$1\r\na\r\n$2\r\nbb\r\n$3\r\nccc\r\n"
	if buf.String() != expected {
		t.Fatalf("expected %q got %q", expected, buf.String())
	}
}

func TestWriteArrayEmpty(t *testing.T) {
	var buf bytes.Buffer
	WriteArray(&buf, []string{})
	if buf.String() != "*0\r\n" {
		t.Fatalf("expected '*0\\r\\n' got %q", buf.String())
	}
}

func TestParseRoundtrip(t *testing.T) {
	commands := [][]string{
		{"SET", "key", "val"},
		{"GET", "key"},
		{"PING"},
		{"DEL", "a", "b", "c"},
		{"HSET", "hash", "f1", "v1", "f2", "v2"},
	}

	for _, cmd := range commands {
		var buf bytes.Buffer
		WriteArray(&buf, cmd)

		reader := bufio.NewReader(&buf)
		parsed, err := Parse(reader)
		if err != nil {
			t.Fatalf("roundtrip parse failed for %v: %v", cmd, err)
		}
		if len(parsed) != len(cmd) {
			t.Fatalf("roundtrip length mismatch for %v: got %d", cmd, len(parsed))
		}
		for i := range cmd {
			if parsed[i] != cmd[i] {
				t.Fatalf("roundtrip arg mismatch for %v: index %d got %q expected %q",
					cmd, i, parsed[i], cmd[i])
			}
		}
	}
}
