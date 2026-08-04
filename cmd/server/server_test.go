package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/imran-binhasan/warpdb/internal/config"
)

func sendCommand(t *testing.T, conn net.Conn, cmd string) string {
	t.Helper()
	if _, err := conn.Write([]byte(cmd)); err != nil {
		t.Fatalf("write error for %q: %v", cmd, err)
	}
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read error for %q: %v", cmd, err)
	}
	return strings.TrimSpace(resp)
}

func sendRESP(t *testing.T, conn net.Conn, args ...string) string {
	t.Helper()
	req := fmt.Sprintf("*%d\r\n", len(args))
	for _, a := range args {
		req += fmt.Sprintf("$%d\r\n%s\r\n", len(a), a)
	}
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write error for %v: %v", args, err)
	}
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read error for %v: %v", args, err)
	}
	return strings.TrimSpace(resp)
}

func startTestServer(t *testing.T) (string, func()) {
	t.Helper()
	cfg := config.Config{
		Port:           0,
		WALPath:        t.TempDir() + "/wal",
		LogLevel:       "error",
		MaxClients:     100,
		WalMaxSizeMB:   1,
		TimeoutSeconds: 30,
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	go Serve(cfg)
	time.Sleep(100 * time.Millisecond)

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	return addr, func() {}
}

func TestServerPing(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendRESP(t, conn, "PING")
	if resp != "+PONG" {
		t.Fatalf("expected +PONG got %s", resp)
	}
}

func TestServerSetGet(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendRESP(t, conn, "SET", "testkey", "hello")
	if resp != "+OK" {
		t.Fatalf("expected +OK got %s", resp)
	}
	resp = sendRESP(t, conn, "GET", "testkey")
	if resp != "$5" {
		t.Fatalf("expected $5 got %s", resp)
	}
}

func TestServerIncr(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sendRESP(t, conn, "SET", "counter", "0")
	resp := sendRESP(t, conn, "INCR", "counter")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
}

func TestServerExists(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sendRESP(t, conn, "SET", "ekey", "val")
	resp := sendRESP(t, conn, "EXISTS", "ekey")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
	resp = sendRESP(t, conn, "EXISTS", "nokey")
	if resp != ":0" {
		t.Fatalf("expected :0 got %s", resp)
	}
}

func TestServerDel(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sendRESP(t, conn, "SET", "dkey", "val")
	resp := sendRESP(t, conn, "DEL", "dkey")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
}

func TestServerListOps(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendRESP(t, conn, "RPUSH", "mylist", "a", "b", "c")
	if resp != ":3" {
		t.Fatalf("expected :3 got %s", resp)
	}
	resp = sendRESP(t, conn, "LPOP", "mylist")
	if resp != "$1" {
		t.Fatalf("expected $1 got %s", resp)
	}
	resp = sendRESP(t, conn, "LLEN", "mylist")
	if resp != ":2" {
		t.Fatalf("expected :2 got %s", resp)
	}
}

func TestServerSetOps(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendRESP(t, conn, "SADD", "myset", "x", "y", "z")
	if resp != ":3" {
		t.Fatalf("expected :3 got %s", resp)
	}
	resp = sendRESP(t, conn, "SISMEMBER", "myset", "x")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
	resp = sendRESP(t, conn, "SCARD", "myset")
	if resp != ":3" {
		t.Fatalf("expected :3 got %s", resp)
	}
}

func TestServerHashOps(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp := sendRESP(t, conn, "HSET", "myhash", "name", "warpdb")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
	resp = sendRESP(t, conn, "HGET", "myhash", "name")
	if resp != "$6" {
		t.Fatalf("expected $6 got %s", resp)
	}
	resp = sendRESP(t, conn, "HEXISTS", "myhash", "name")
	if resp != ":1" {
		t.Fatalf("expected :1 got %s", resp)
	}
}

func TestServerDBSize(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sendRESP(t, conn, "SET", "a", "1")
	sendRESP(t, conn, "SET", "b", "2")
	resp := sendRESP(t, conn, "DBSIZE")
	if resp != ":2" {
		t.Fatalf("expected :2 got %s", resp)
	}
}

func TestServerInfo(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.Write([]byte("*1\r\n$4\r\nINFO\r\n"))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "$") {
		t.Fatalf("expected bulk string response, got %s", line)
	}
}
