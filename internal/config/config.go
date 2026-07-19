package config

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"time"
)

type Config struct {
	Port     int    `json:"port"`
	WALPath  string `json:"wal_path"`
	LogLevel string `json:"log_level"`

	RequirePass string `json:"requirepass"`

	MaxClients int `json:"maxclients"`

	WalMaxSizeMB    int  `json:"wal_max_size_mb"`
	WalAutoCompact  bool `json:"wal_auto_compact"`

	TimeoutSeconds int `json:"timeout_seconds"`

	MaxMemoryMB int `json:"maxmemory_mb"`

	Databases int `json:"databases"`
}

func (c Config) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

func Default() Config {
	return Config{
		Port:            6379,
		WALPath:         "wal.log",
		LogLevel:        "info",
		MaxClients:      10000,
		WalMaxSizeMB:    64,
		WalAutoCompact:  true,
		TimeoutSeconds:  300,
		MaxMemoryMB:     0,
		Databases:       16,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ParseFlags() (Config, string) {
	cfg := Default()
	var configPath string
	flag.IntVar(&cfg.Port, "port", cfg.Port, "server port")
	flag.StringVar(&cfg.WALPath, "wal", cfg.WALPath, "WAL file path")
	flag.StringVar(&cfg.RequirePass, "requirepass", cfg.RequirePass, "password for AUTH")
	flag.IntVar(&cfg.MaxClients, "maxclients", cfg.MaxClients, "max concurrent clients")
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()
	return cfg, configPath
}

func Merge(cli Config, file Config) Config {
	cfg := file

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			cfg.Port = cli.Port
		case "wal":
			cfg.WALPath = cli.WALPath
		case "requirepass":
			cfg.RequirePass = cli.RequirePass
		case "maxclients":
			cfg.MaxClients = cli.MaxClients
		}
	})

	return cfg
}

func LogLevelFromString(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
