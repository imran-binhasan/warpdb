# WarpDB

A production-ready, Redis-compatible, hybrid in-memory/persistent key-value store written in Go from scratch.
Connects to any Redis client with no changes — `redis-cli`, `ioredis`, `redis-py`, all work out of the box.

```bash
go run ./cmd/main.go
redis-cli -p 6379 SET name Imran
redis-cli -p 6379 GET name
```

---

## Architecture

```
TCP client (RESP)
      │
      ▼
cmd/server              — TCP listener :6379, one goroutine per connection
      │
      ├── core/protocol     — RESP parser and response writers
      └── core/commands     — command dispatch (SET, GET, DEL, INCR, ...)
                 │
                 ▼
         engine.StorageEngine        — pluggable interface
                 │
                 ├── WALEngine       — default: logs writes, then applies to memory
                 │       ├── wal/WAL         — binary append-only log with batched fsync & CRC32
                 │       └── MemoryEngine    — in-memory delegate
                 │
                 └── MemoryEngine    — standalone, no persistence
                         └── core/store.Store  — 256-shard concurrent hashmap
```

### Key design decisions

**256-shard concurrent hashmap**  
Keys are routed to one of 256 shards via FNV-1a hash (`hash & 255`).
Each shard has its own `sync.RWMutex` — contention drops ~256x compared to a single
global lock. Goroutines hitting different keys operate on different shards in parallel.

**Binary Write-Ahead Log with CRC32 checksums**  
Every write is logged in a compact binary format with CRC32 integrity checks before it
touches memory. A background flusher goroutine collects all concurrent writes between
wakeups into one batch, writes them all, calls `fsync` once, then signals every
waiting goroutine simultaneously. One `fsync` per batch instead of one per write —
same durability guarantee, orders of magnitude less IO overhead.

**WAL segment rotation and compaction**  
WAL segments are automatically rotated when they exceed a configurable size threshold.
Compaction removes overwritten and deleted entries to reclaim disk space.

**Two-phase locking on reads**  
`Get` and `Exists` take a read lock (phase 1) to copy the value and expiry, then release
it immediately. Only if the key is expired do they re-acquire a write lock (phase 2) to
delete it — and re-check under the write lock to prevent a race where another goroutine
re-set the key between phases. The common path — live key — never takes a write lock.

**Dual-strategy TTL eviction**  
Lazy expiry: every read checks expiry before returning a value.  
Active expiry: a background goroutine sweeps all 256 shards every 100ms, deleting expired
keys even if nobody reads them. Matches Redis `EXPIRE`/`TTL`/`PERSIST` semantics exactly.

**Zero allocations on the hot path**  
`Set` and `Get` cause zero heap allocations when operating on existing keys —
confirmed by `go test -bench -benchmem`. No GC pressure under sustained load.

---

## Supported Commands

### String Commands
| Command | Syntax | Description |
|---|---|---|
| `PING` | `PING [message]` | Liveness check, echoes message if provided |
| `SET` | `SET key value` | Set a key, clears any existing TTL |
| `GET` | `GET key` | Get a value, returns nil if missing or expired |
| `DEL` | `DEL key [key ...]` | Delete one or more keys |
| `EXISTS` | `EXISTS key` | Returns 1 if key exists, 0 otherwise |
| `INCR` | `INCR key` | Increment integer value, initialises to 0 if missing |
| `DECR` | `DECR key` | Decrement integer value, initialises to 0 if missing |
| `INCRBY` | `INCRBY key increment` | Increment by given amount |
| `DECRBY` | `DECRBY key decrement` | Decrement by given amount |
| `APPEND` | `APPEND key value` | Append value to key |
| `STRLEN` | `STRLEN key` | Length of string value |
| `MSET` | `MSET key val [key val ...]` | Set multiple keys |
| `MGET` | `MGET key [key ...]` | Get multiple keys |
| `RENAME` | `RENAME key newkey` | Rename a key |
| `TYPE` | `TYPE key` | Get type of key (string/list/set/hash/none) |
| `RANDOMKEY` | `RANDOMKEY` | Get a random key |
| `EXPIRE` | `EXPIRE key seconds` | Set TTL in seconds |
| `TTL` | `TTL key` | Remaining TTL in seconds (-1 = no expiry, -2 = missing) |
| `PERSIST` | `PERSIST key` | Remove TTL, make key permanent |
| `KEYS` | `KEYS pattern` | List keys matching glob pattern (`*`, `?`, `[abc]`, `[a-z]`) |
| `FLUSHALL` | `FLUSHALL` | Delete all keys across all shards |
| `DBSIZE` | `DBSIZE` | Total number of keys |

### List Commands
| Command | Syntax | Description |
|---|---|---|
| `LPUSH` | `LPUSH key element [element ...]` | Prepend elements to list |
| `RPUSH` | `RPUSH key element [element ...]` | Append elements to list |
| `LPOP` | `LPOP key` | Remove and return first element |
| `RPOP` | `RPOP key` | Remove and return last element |
| `LLEN` | `LLEN key` | List length |
| `LRANGE` | `LRANGE key start stop` | Range of elements |
| `LINDEX` | `LINDEX key index` | Element at index (supports negative) |

### Set Commands
| Command | Syntax | Description |
|---|---|---|
| `SADD` | `SADD key member [member ...]` | Add members to set |
| `SREM` | `SREM key member [member ...]` | Remove members from set |
| `SMEMBERS` | `SMEMBERS key` | All members of set |
| `SISMEMBER` | `SISMEMBER key member` | Check membership |
| `SCARD` | `SCARD key` | Set cardinality |
| `SPOP` | `SPOP key` | Remove and return random member |
| `SRANDMEMBER` | `SRANDMEMBER key` | Get random member without removing |

### Hash Commands
| Command | Syntax | Description |
|---|---|---|
| `HSET` | `HSET key field value [field value ...]` | Set field(s) in hash |
| `HGET` | `HGET key field` | Get field value |
| `HDEL` | `HDEL key field [field ...]` | Delete field(s) from hash |
| `HGETALL` | `HGETALL key` | All fields and values |
| `HKEYS` | `HKEYS key` | All field names |
| `HVALS` | `HVALS key` | All field values |
| `HEXISTS` | `HEXISTS key field` | Check field existence |
| `HLEN` | `HLEN key` | Number of fields |
| `HMSET` | `HMSET key field val [field val ...]` | Set multiple fields |
| `HMGET` | `HMGET key field [field ...]` | Get multiple field values |

### Server Commands
| Command | Syntax | Description |
|---|---|---|
| `AUTH` | `AUTH password` | Authenticate (required if `requirepass` set) |
| `INFO` | `INFO` | Server statistics and metrics |
| `CONFIG GET` | `CONFIG GET parameter` | Get configuration values |
| `CONFIG SET` | `CONFIG SET parameter value` | Set runtime configuration |
| `COMMAND` | `COMMAND` | Command introspection |

---

## Getting Started

**Requirements:** Go 1.21+, no external dependencies

### Local Development

```bash
git clone https://github.com/imran-binhasan/warpdb
cd warpdb

# Build
make build

# Run
go run ./cmd/main.go

# Run with config file
go run ./cmd/main.go -config warpdb.json

# Run with custom port
go run ./cmd/main.go -port 6380

# Run with auth
go run ./cmd/main.go -requirepass secret123
```

### Docker

```bash
# Build and run
docker compose up -d

# Dev mode (no auth, port 6380)
docker compose --profile dev up -d

# Connect
redis-cli -p 6379
```

### Kubernetes

```bash
kubectl apply -f deploy/warpdb.yaml
```

---

## Configuration

Configuration is loaded from a JSON file (default: `warpdb.json` in the working directory).

| Key | Type | Default | Description |
|---|---|---|---|
| `port` | int | 6379 | TCP listen port |
| `wal_path` | string | `wal` | WAL segment directory |
| `log_level` | string | `info` | debug, info, warn, error |
| `requirepass` | string | `""` | Password for AUTH (empty = no auth) |
| `maxclients` | int | 10000 | Maximum concurrent connections |
| `wal_max_size_mb` | int | 64 | Max WAL segment size before rotation |
| `wal_auto_compact` | bool | true | Enable automatic WAL compaction |
| `timeout_seconds` | int | 300 | Client idle timeout (0 = no timeout) |
| `maxmemory_mb` | int | 0 | Memory limit in MB (0 = unlimited) |
| `databases` | int | 16 | Number of logical databases |

CLI flags override file config:
```
-wal <path>  -port <n>  -requirepass <pw>  -maxclients <n>  -config <path>
```

---

## Crash Recovery

WarpDB writes every mutation to a binary Write-Ahead Log with CRC32 checksums
before applying it to memory. On restart the WAL is replayed to rebuild state:

```bash
go run ./cmd/main.go &
redis-cli -p 6379 SET counter 42
redis-cli -p 6379 INCR counter      # → 43
pkill -f warpdb
go run ./cmd/main.go &
redis-cli -p 6379 GET counter       # → "43" — survived restart
```

WAL segments are stored in the `wal/` directory with numbered segment files
and a `manifest.json` tracking active segments. Corruption is detected via
CRC32 checksums on every entry.

---

## Benchmarks

Run on **Intel i5-10300H · Linux · Go 1.26 · 256 shards**  
Keyspace: 100k keys · Value size: 64 bytes · Benchtime: 5s

```
make bench
```

![Benchmark results](benchmark.png)

**Isolated operations**

| Operation | 1 core | 2 cores | 4 cores | 8 cores | Latency |
|---|---|---|---|---|---|
| GET | 2.87M/s | 4.20M/s | 6.74M/s | 7.55M/s | 349 ns |
| SET | 2.31M/s | 3.42M/s | 5.56M/s | 6.61M/s | 432 ns |
| INCR | 2.81M/s | 3.62M/s | 5.22M/s | 5.84M/s | 356 ns |
| DEL | 2.16M/s | 2.99M/s | 4.73M/s | 5.26M/s | 463 ns |

**Mixed workloads**

| Workload | Ratio | 1 core | 4 cores | 8 cores |
|---|---|---|---|---|
| Read-heavy | 95% GET / 5% SET | 2.38M/s | 3.88M/s | 5.02M/s |
| Balanced | 50% GET / 50% SET | 1.77M/s | 3.43M/s | 5.53M/s |
| Write-heavy | 20% GET / 80% SET | 2.39M/s | 4.69M/s | 5.18M/s |

**Hot path allocations**

| Operation | allocs/op | B/op | ns/op |
|---|---|---|---|
| SET | 0 | 0 | 69.92 |
| GET | 0 | 0 | 54.13 |

---

## Project Structure

```
warpdb/
├── cmd/
│   ├── main.go                  — entry point with config + flag parsing
│   ├── server/
│   │   ├── server.go            — TCP listener, graceful shutdown, auth middleware
│   │   └── server_test.go       — integration tests (real TCP server)
├── core/
│   ├── commands/
│   │   └── commands.go          — command dispatch (50+ commands)
│   ├── protocol/
│   │   ├── protocol.go          — RESP parser and response writers
│   │   └── protocol_test.go     — RESP parsing/serialization tests
│   └── store/
│       ├── store.go             — 256-shard hashmap, lists, sets, hashes, TTL
│       └── store_test.go        — unit tests and benchmarks
├── engine/
│   ├── engine.go                — StorageEngine interface
│   ├── memory.go                — pure in-memory engine
│   └── wal_engine.go            — WAL-backed engine with persistence
├── internal/
│   ├── config/
│   │   ├── config.go            — configuration struct, JSON loader, CLI flags
│   │   └── config_test.go       — config loading tests
│   └── stats/
│       └── stats.go             — atomic server statistics for INFO command
├── wal/
│   ├── wal.go                   — binary WAL with batched fsync, segment rotation
│   ├── format.go                — binary entry encoding/decoding, CRC32
│   └── format_test.go           — binary format tests
├── scripts/
│   └── fmtbench.awk             — benchmark output formatter
├── Dockerfile                   — multi-stage production Docker image
├── docker-compose.yml           — Docker Compose for dev and production
├── .gitignore
├── warpdb.json                  — sample configuration file
├── Makefile
└── go.mod                       — zero external dependencies
```

---

## Production Deployment

### System Requirements

- Linux (amd64/arm64) or macOS
- 512MB+ RAM (depends on dataset size)
- Go 1.21+ (if building from source)

### Security

- Set `requirepass` in config for authentication
- Run behind a firewall; WarpDB has no TLS natively
- Use a reverse proxy (nginx, HAProxy) for TLS termination
- Run as a non-root user (Docker image does this by default)

### Persistence

- WAL segments are stored in the `wal_path` directory
- Set `wal_max_size_mb` to control segment rotation (default: 64MB)
- WAL compaction can be triggered to reclaim disk space
- Mount `wal_path` to a persistent volume in containerized deployments

### Monitoring

- `INFO` command provides server stats, client counts, memory usage, keyspace info
- The `/health` endpoint can be checked with `PING`
- Docker image includes a HEALTHCHECK

### Performance Tuning

- Increase `maxclients` for high-concurrency workloads (default: 10000)
- Increase OS file descriptor limits (`ulimit -n 65536`)
- Set `net.core.somaxconn=65535` for high connection rates
- Set `timeout_seconds` to disconnect idle clients (default: 300)

---

## Make Targets

```
make build    — compile
make test     — run all tests  
make race     — run tests with Go race detector
make bench    — full benchmark suite with formatted output
make clean    — remove build artifacts and wal.log
```

---

## What's Next

- Replication and clustering support
- Pub/Sub messaging
- Lua scripting
- Transactions (MULTI/EXEC)
- Eviction policies (LRU, LFU)
- TLS support
- Redis Sentinel compatibility
