package server
import (
	"bufio"
	"fmt"
	"net"
	"log"
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
}