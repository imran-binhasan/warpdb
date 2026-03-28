package commands

import (
	"io"
	"strings"

	"github.com/imran-binhasan/warpdb/core/protocol"
	// "github.com/imran-binhasan/warpdb/core/store"
	"github.com/imran-binhasan/warpdb/engine"
)

func Handle(args []string, engine engine.StorageEngine, w io.Writer) {

	if len(args) == 0 {
		protocol.WriteError(w, "ERR empty command")
		return
	}

	switch strings.ToUpper(args[0]) {

	case "PING":
		if len(args) == 1 {
			protocol.WriteSimpleString(w, "PONG")
		} else {
			protocol.WriteBulkString(w, args[1])
		}

	case "SET":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for SET")
			return
		}
		engine.Set(args[1], args[2])
		protocol.WriteSimpleString(w, "OK")

	case "GET":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for GET")
			return
		}
		res, err := engine.Get(args[1])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, res)

	case "DEL":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for DEL")
			return
		}
		err := engine.Del(args[1])
		if err != nil {
			protocol.WriteInteger(w, 0)
			return
		}
		protocol.WriteInteger(w, 1)
	case "EXISTS":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for EXISTS")
			return
		}
		if engine.Exists(args[1]) {
			protocol.WriteInteger(w, 1)
		} else {
			protocol.WriteInteger(w, 0)
		}

	case "INCR":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for INCR")
			return
		}
		res, err := engine.Incr(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		protocol.WriteInteger(w, res)

	case "DECR":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for DECR")
			return
		}
		res, err := engine.Decr(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		protocol.WriteInteger(w, res)

	default:
		protocol.WriteError(w, "ERR unknown command")
	}

}
