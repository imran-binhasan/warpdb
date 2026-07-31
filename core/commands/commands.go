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
	cmd := strings.ToUpper(args[0])

	switch cmd {
	case "PING":
		if len(args) == 1 {
			protocol.WriteSimpleString(w, "PONG")
		} else {
			protocol.WriteBulkString(w, args[1])
		}

	case "INFO":
		protocol.WriteBulkString(w, h.stats.InfoSection(h.engine.Size()))

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
		if len(args) < 3 {
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
		if len(args) < 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for DEL")
			return
		}
		deleted := 0
		for _, key := range args[1:] {
			if err := h.engine.Del(key); err == nil {
				deleted++
			}
		}
		protocol.WriteInteger(w, deleted)

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

	case "INCRBY":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for INCRBY")
			return
		}
		incr, err := strconv.Atoi(args[2])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		var result int
		for i := 0; i < incr; i++ {
			result, err = h.engine.Incr(args[1])
			if err != nil {
				protocol.WriteError(w, "ERR value is not an integer or out of range")
				return
			}
		}
		protocol.WriteInteger(w, result)

	case "DECRBY":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for DECRBY")
			return
		}
		decr, err := strconv.Atoi(args[2])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer or out of range")
			return
		}
		var result int
		for i := 0; i < decr; i++ {
			result, err = h.engine.Decr(args[1])
			if err != nil {
				protocol.WriteError(w, "ERR value is not an integer or out of range")
				return
			}
		}
		protocol.WriteInteger(w, result)

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
		if err := h.engine.Expire(args[1], time.Duration(secs)*time.Second); err != nil {
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
		if err := h.engine.Persist(args[1]); err != nil {
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

	case "RENAME":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for RENAME")
			return
		}
		val, err := h.engine.Get(args[1])
		if err != nil {
			protocol.WriteError(w, "ERR no such key")
			return
		}
		h.engine.Set(args[2], val)
		h.engine.Del(args[1])
		protocol.WriteSimpleString(w, "OK")

	case "TYPE":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for TYPE")
			return
		}
		if _, err := h.engine.Get(args[1]); err == nil {
			protocol.WriteSimpleString(w, "string")
		} else if h.engine.LLen(args[1]) > 0 || h.engine.Exists(args[1]) {
			protocol.WriteSimpleString(w, "list")
		} else if h.engine.SCard(args[1]) > 0 || h.engine.Exists(args[1]) {
			protocol.WriteSimpleString(w, "set")
		} else if h.engine.HLen(args[1]) > 0 || h.engine.Exists(args[1]) {
			protocol.WriteSimpleString(w, "hash")
		} else {
			protocol.WriteSimpleString(w, "none")
		}

	case "APPEND":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for APPEND")
			return
		}
		existing, err := h.engine.Get(args[1])
		if err != nil {
			h.engine.Set(args[1], args[2])
			protocol.WriteInteger(w, len(args[2]))
			return
		}
		newVal := existing + args[2]
		h.engine.Set(args[1], newVal)
		protocol.WriteInteger(w, len(newVal))

	case "STRLEN":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for STRLEN")
			return
		}
		val, err := h.engine.Get(args[1])
		if err != nil {
			protocol.WriteInteger(w, 0)
			return
		}
		protocol.WriteInteger(w, len(val))

	case "MSET":
		if len(args) < 3 || len(args)%2 != 1 {
			protocol.WriteError(w, "ERR wrong number of arguments for MSET")
			return
		}
		for i := 1; i < len(args); i += 2 {
			h.engine.Set(args[i], args[i+1])
		}
		protocol.WriteSimpleString(w, "OK")

	case "MGET":
		if len(args) < 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for MGET")
			return
		}
		result := make([]string, 0, len(args)-1)
		for _, key := range args[1:] {
			val, err := h.engine.Get(key)
			if err != nil {
				result = append(result, "")
			} else {
				result = append(result, val)
			}
		}
		protocol.WriteArray(w, result)

	case "RANDOMKEY":
		keys := h.engine.Keys("*")
		if len(keys) == 0 {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, keys[0])

	// --- List Commands ---
	case "LPUSH":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for LPUSH")
			return
		}
		count := h.engine.LPush(args[1], args[2:]...)
		protocol.WriteInteger(w, count)

	case "RPUSH":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for RPUSH")
			return
		}
		count := h.engine.RPush(args[1], args[2:]...)
		protocol.WriteInteger(w, count)

	case "LPOP":
		if len(args) < 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for LPOP")
			return
		}
		val, err := h.engine.LPop(args[1])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	case "RPOP":
		if len(args) < 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for RPOP")
			return
		}
		val, err := h.engine.RPop(args[1])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	case "LLEN":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for LLEN")
			return
		}
		protocol.WriteInteger(w, h.engine.LLen(args[1]))

	case "LRANGE":
		if len(args) != 4 {
			protocol.WriteError(w, "ERR wrong number of arguments for LRANGE")
			return
		}
		start, err := strconv.Atoi(args[2])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer")
			return
		}
		stop, err := strconv.Atoi(args[3])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer")
			return
		}
		protocol.WriteArray(w, h.engine.LRange(args[1], start, stop))

	case "LINDEX":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for LINDEX")
			return
		}
		index, err := strconv.Atoi(args[2])
		if err != nil {
			protocol.WriteError(w, "ERR value is not an integer")
			return
		}
		val, err := h.engine.LIndex(args[1], index)
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	// --- Set Commands ---
	case "SADD":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for SADD")
			return
		}
		protocol.WriteInteger(w, h.engine.SAdd(args[1], args[2:]...))

	case "SREM":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for SREM")
			return
		}
		protocol.WriteInteger(w, h.engine.SRem(args[1], args[2:]...))

	case "SMEMBERS":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for SMEMBERS")
			return
		}
		protocol.WriteArray(w, h.engine.SMembers(args[1]))

	case "SISMEMBER":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for SISMEMBER")
			return
		}
		if h.engine.SIsMember(args[1], args[2]) {
			protocol.WriteInteger(w, 1)
		} else {
			protocol.WriteInteger(w, 0)
		}

	case "SCARD":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for SCARD")
			return
		}
		protocol.WriteInteger(w, h.engine.SCard(args[1]))

	case "SPOP":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for SPOP")
			return
		}
		val, err := h.engine.SPop(args[1])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	case "SRANDMEMBER":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for SRANDMEMBER")
			return
		}
		val, err := h.engine.SRandMember(args[1])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	// --- Hash Commands ---
	case "HSET":
		if len(args) < 4 || (len(args)-2)%2 != 0 {
			protocol.WriteError(w, "ERR wrong number of arguments for HSET")
			return
		}
		count := 0
		for i := 2; i < len(args); i += 2 {
			count += h.engine.HSet(args[1], args[i], args[i+1])
		}
		protocol.WriteInteger(w, count)

	case "HGET":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for HGET")
			return
		}
		val, err := h.engine.HGet(args[1], args[2])
		if err != nil {
			protocol.WriteNull(w)
			return
		}
		protocol.WriteBulkString(w, val)

	case "HDEL":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for HDEL")
			return
		}
		protocol.WriteInteger(w, h.engine.HDel(args[1], args[2:]...))

	case "HGETALL":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for HGETALL")
			return
		}
		protocol.WriteArray(w, h.engine.HGetAll(args[1]))

	case "HKEYS":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for HKEYS")
			return
		}
		protocol.WriteArray(w, h.engine.HKeys(args[1]))

	case "HVALS":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for HVALS")
			return
		}
		protocol.WriteArray(w, h.engine.HVals(args[1]))

	case "HEXISTS":
		if len(args) != 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for HEXISTS")
			return
		}
		if h.engine.HExists(args[1], args[2]) {
			protocol.WriteInteger(w, 1)
		} else {
			protocol.WriteInteger(w, 0)
		}

	case "HLEN":
		if len(args) != 2 {
			protocol.WriteError(w, "ERR wrong number of arguments for HLEN")
			return
		}
		protocol.WriteInteger(w, h.engine.HLen(args[1]))

	case "HMSET":
		if len(args) < 4 || (len(args)-2)%2 != 0 {
			protocol.WriteError(w, "ERR wrong number of arguments for HMSET")
			return
		}
		fields := make(map[string]string)
		for i := 2; i < len(args); i += 2 {
			fields[args[i]] = args[i+1]
		}
		h.engine.HMSet(args[1], fields)
		protocol.WriteSimpleString(w, "OK")

	case "HMGET":
		if len(args) < 3 {
			protocol.WriteError(w, "ERR wrong number of arguments for HMGET")
			return
		}
		protocol.WriteArray(w, h.engine.HMGet(args[1], args[2:]...))

	default:
		protocol.WriteError(w, fmt.Sprintf("ERR unknown command '%s'", args[0]))
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
