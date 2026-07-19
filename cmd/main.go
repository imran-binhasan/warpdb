package main

import (
	"log/slog"
	"os"

	"github.com/imran-binhasan/warpdb/cmd/server"
	"github.com/imran-binhasan/warpdb/internal/config"
)

func main() {
	cliCfg, configPath := config.ParseFlags()
	fileCfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	cfg := config.Merge(cliCfg, fileCfg)

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevelFromString(cfg.LogLevel),
	})))

	server.Serve(cfg)
}
