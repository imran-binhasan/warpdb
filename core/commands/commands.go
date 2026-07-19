package commands

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/engine"
)

type Handler struct {
	engine engine.StorageEngine
}

func NewHandler(eng engine.StorageEngine) *Handler {
	return &Handler{engine: eng}
}

func (h *Handler) Handle(args []string, w io.Writer) {
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
		h.engine.Set(args[1], args[2])
		protocol.WriteSimpleString(w, "OK")

	case "GET":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for GET")
			return
		}
		res, err := h.engine.Get(args[1])
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
		err := h.engine.Del(args[1])
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
		if h.engine.Exists(args[1]) {
			protocol.WriteInteger(w, 1)
		} else {
			protocol.WriteInteger(w, 0)
		}

	case "INCR":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for INCR")
			return
		}
		res, err := h.engine.Incr(args[1])
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
		res, err := h.engine.Decr(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		protocol.WriteInteger(w, res)

	case "EXPIRE":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for EXPIRE")
			return
		}
		secs, err := strconv.Atoi(args[2])
		if err != nil || secs < 0 {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		err = h.engine.Expire(args[1], time.Duration(secs)*time.Second)
		if err != nil {
			protocol.WriteInteger(w, 0)
			return
		}
		protocol.WriteInteger(w, 1)

	case "TTL":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for TTL")
			return
		}
		remaining, _ := h.engine.TTL(args[1])
		if remaining == -2 {
			protocol.WriteInteger(w, -2)
			return
		}
		if remaining == -1 {
			protocol.WriteInteger(w, -1)
			return
		}
		protocol.WriteInteger(w, int(remaining.Seconds()))

	case "PERSIST":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for PERSIST")
			return
		}
		err := h.engine.Persist(args[1])
		if err != nil {
			protocol.WriteInteger(w, 0)
			return
		}
		protocol.WriteInteger(w, 1)

	case "KEYS":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for KEYS")
			return
		}
		keys := h.engine.Keys(args[1])
		protocol.WriteArray(w, keys)

	case "FLUSHALL":
		h.engine.FlushAll()
		protocol.WriteSimpleString(w, "OK")

	default:
		protocol.WriteError(w, "ERR unknown command")
	}
}
