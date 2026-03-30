package server

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/imran-binhasan/warpdb/core/commands"
	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/engine"
)

func Server() {
	engine, err := engine.NewWALEngine("wal.log")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("WarpDB listening on :6379")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error", err)
			continue
		}
		go handleClient(conn, engine)
	}
}

func handleClient(conn net.Conn, engine engine.StorageEngine) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		args, err := protocol.Parse(reader)
		if err != nil {
			log.Println("Parse error", err)
			break
		}
		commands.Handle(args, engine, conn)
	}
}
