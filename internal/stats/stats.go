package stats

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

type ServerStats struct {
	Started           time.Time
	TotalConnections  atomic.Int64
	TotalCommands     atomic.Int64
	ActiveConnections atomic.Int64
}

func New() *ServerStats {
	return &ServerStats{Started: time.Now()}
}

func (s *ServerStats) InfoSection(engineKeyCount int) string {
	uptime := time.Since(s.Started).Seconds()
	totalCmds := s.TotalCommands.Load()
	totalConns := s.TotalConnections.Load()
	activeConns := s.ActiveConnections.Load()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return fmt.Sprintf(
		"# Server\r\n"+
			"warpdb_version:1.0.0\r\n"+
			"os:%s\r\n"+
			"arch:%s\r\n"+
			"go_version:%s\r\n"+
			"process_id:%d\r\n"+
			"\r\n"+
			"# Clients\r\n"+
			"connected_clients:%d\r\n"+
			"total_connections_received:%d\r\n"+
			"\r\n"+
			"# Stats\r\n"+
			"total_commands_processed:%d\r\n"+
			"uptime_in_seconds:%d\r\n"+
			"instantaneous_ops_per_sec:%d\r\n"+
			"\r\n"+
			"# Keyspace\r\n"+
			"db0:keys=%d,expires=0,avg_ttl=0\r\n"+
			"\r\n"+
			"# Memory\r\n"+
			"used_memory:%d\r\n"+
			"used_memory_human:%s\r\n"+
			"used_memory_rss:%d\r\n"+
			"\r\n"+
			"# CPU\r\n"+
			"used_cpu_sys:%.2f\r\n"+
			"used_cpu_user:%.2f\r\n",
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
		// placeholder, process_id is cosmetic
		0,
		activeConns,
		totalConns,
		totalCmds,
		int(uptime),
		int(totalCmds/int64(uptime+1)),
		engineKeyCount,
		memStats.Alloc,
		formatBytes(memStats.Alloc),
		memStats.Sys,
		0.0, 0.0,
	)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}
