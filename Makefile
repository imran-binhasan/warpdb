.PHONY: build test bench race clean

build:
	@go build ./...
	@echo "build ok"

test:
	@go test ./...

race:
	@go test -race ./...

bench:
	@echo ""
	@echo "=== WarpDB Benchmarks ==="
	@echo "    cpu: $(shell go env GOARCH) · shards: 256 · benchtime: 5s"
	@echo ""
	@echo "--- isolated operations ---"
	@go test -bench='Benchmark(Set|Get|Incr|Del)$$' \
		-benchmem -benchtime=5s -cpu=1,2,4,8 \
		./core/store/ | grep -E '^Benchmark'
	@echo ""
	@echo "--- mixed workloads (ReadHeavy=95/5  Balanced=50/50  WriteHeavy=80/20) ---"
	@go test -bench='Benchmark(ReadHeavy|Balanced|WriteHeavy)' \
		-benchmem -benchtime=5s -cpu=1,2,4,8 \
		./core/store/ | grep -E '^Benchmark'
	@echo ""
	@echo "--- hot path allocations ---"
	@go test -bench='Benchmark(Set|Get)Allocs' \
		-benchmem -benchtime=5s \
		./core/store/ | grep -E '^Benchmark'
	@echo ""
	@echo "--- shard distribution (100k keys → 256 shards) ---"
	@go test -bench='BenchmarkShardDistribution$$' \
		-benchmem -benchtime=1s \
		./core/store/ | grep -E '^Benchmark'
	@echo ""

clean:
	@go clean ./...
	@rm -f wal.log
	@echo "clean ok"