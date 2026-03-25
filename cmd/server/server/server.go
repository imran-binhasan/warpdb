package server

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/imran-binhasan/warpdb/core/commands"
	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/core/store"
)

func Server(){
	s := store.NewStore()
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
		go handleClient(conn, s)
	}
}

func handleClient(conn net.Conn, s *store.Store) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		args, err := protocol.Parse(reader)
		if err != nil {
			log.Println("Parse error", err)
		break
		} 
		commands.Handle(args, s, conn)
	}
}