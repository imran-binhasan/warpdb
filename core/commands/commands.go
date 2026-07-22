package commands

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/engine"
	"github.com/imran-binhasan/warpdb/internal/config"
	"github.com/imran-binhasan/warpdb/internal/stats"
)

type Handler struct {
	engine engine.StorageEngine
	cfg    *config.Config
	stats  *stats.ServerStats
}

func NewHandler(eng engine.StorageEngine, cfg *config.Config, st *stats.ServerStats) *Handler {
	return &Handler{engine: eng, cfg: cfg, stats: st}
}

func (h *Handler) Handle(args []string, w io.Writer) {
	if len(args) == 0 {
		protocol.WriteError(w, "ERR empty command")
		return
	}

	h.stats.TotalCommands.Add(1)

	switch strings.ToUpper(args[0]) {
	case "PING":
		if len(args) == 1 {
			protocol.WriteSimpleString(w, "PONG")
		} else {
			protocol.WriteBulkString(w, args[1])
		}

	case "INFO":
		section := h.stats.InfoSection(h.engine.Size())
		protocol.WriteBulkString(w, section)

	case "CONFIG":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for CONFIG")
			return
		}
		switch strings.ToUpper(args[1]) {
		case "GET":
			h.configGet(args[2:], w)
		case "SET":
			if len(args) < 4 {
				protocol.WriteError(w, "ERR wrong number of arguments for CONFIG SET")
				return
			}
			h.configSet(args[2], args[3], w)
		default:
			protocol.WriteError(w, "ERR CONFIG subcommand must be GET or SET")
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

	case "DBSIZE":
		protocol.WriteInteger(w, h.engine.Size())

	case "COMMAND":
		protocol.WriteSimpleString(w, "OK")

	default:
		protocol.WriteError(w, "ERR unknown command")
	}
}

func (h *Handler) configGet(patterns []string, w io.Writer) {
	cfgMap := map[string]string{
		"port":             fmt.Sprintf("%d", h.cfg.Port),
		"wal_path":         h.cfg.WALPath,
		"log_level":        h.cfg.LogLevel,
		"requirepass":      h.cfg.RequirePass,
		"maxclients":       fmt.Sprintf("%d", h.cfg.MaxClients),
		"timeout":          fmt.Sprintf("%d", h.cfg.TimeoutSeconds),
		"maxmemory":        fmt.Sprintf("%d", h.cfg.MaxMemoryMB),
		"databases":        fmt.Sprintf("%d", h.cfg.Databases),
		"wal_max_size_mb":  fmt.Sprintf("%d", h.cfg.WalMaxSizeMB),
		"wal_auto_compact": fmt.Sprintf("%t", h.cfg.WalAutoCompact),
	}

	var result []string
	for _, pattern := range patterns {
		for k, v := range cfgMap {
			if matchSimple(pattern, k) {
				result = append(result, k, v)
			}
		}
	}
	protocol.WriteArray(w, result)
}

func (h *Handler) configSet(key, value string, w io.Writer) {
	switch strings.ToLower(key) {
	case "requirepass":
		h.cfg.RequirePass = value
	case "maxclients":
		v, err := strconv.Atoi(value)
		if err != nil || v <= 0 {
			protocol.WriteError(w, "ERR invalid value")
			return
		}
		h.cfg.MaxClients = v
	case "timeout":
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 {
			protocol.WriteError(w, "ERR invalid value")
			return
		}
		h.cfg.TimeoutSeconds = v
	case "log_level":
		if value != "debug" && value != "info" && value != "warn" && value != "error" {
			protocol.WriteError(w, "ERR invalid log level")
			return
		}
		h.cfg.LogLevel = value
	default:
		protocol.WriteError(w, "ERR Unsupported CONFIG parameter")
		return
	}
	protocol.WriteSimpleString(w, "OK")
}

func matchSimple(pattern, key string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(pattern, key)
}
