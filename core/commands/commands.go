package commands

import (
	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/core/store"
	"io"
	"strings"
)

func Handle(args []string, store *store.Store, w io.Writer) {

	if len(args) == 0 {
		protocol.WriteError(w, "ERR empty command")
		return
	}

	switch strings.ToUpper(args[0]) {

	case "SET":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for SET")
			return
		}
		store.Set(args[1], args[2])
		protocol.WriteSimpleString(w, "OK")

	case "GET":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for GET")
			return
		}
		res, err := store.Get(args[1])
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
		err := store.Del(args[1])
		if err != nil {
			protocol.WriteInteger(w, 0)
			return
		}
		protocol.WriteInteger(w, 1)

	case "INCR":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for INCR")
			return
		}
		res, err := store.Incr(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer")
			return
		}
		protocol.WriteInteger(w, res)

	case "DECR":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for DECR")
			return
		}
		res, err := store.Decr(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer")
			return
		}
		protocol.WriteInteger(w, res)

	default:
		protocol.WriteError(w, "ERR unknown command")
	}

}
