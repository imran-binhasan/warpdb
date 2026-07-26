package server

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/imran-binhasan/warpdb/core/commands"
	"github.com/imran-binhasan/warpdb/core/protocol"
	"github.com/imran-binhasan/warpdb/engine"
	"github.com/imran-binhasan/warpdb/internal/config"
	"github.com/imran-binhasan/warpdb/internal/stats"
)

type Server struct {
	cfg    config.Config
	engine engine.StorageEngine
	ln     net.Listener
	sem    chan struct{}
	stats  *stats.ServerStats
}

func Serve(cfg config.Config) {
	eng, err := engine.NewWALEngine(cfg.WALPath, cfg.WalMaxSizeMB)
	if err != nil {
		slog.Error("failed to initialize storage engine", "error", err)
		os.Exit(1)
	}
	slog.Info("storage engine initialized", "wal", cfg.WALPath)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	srv := &Server{
		cfg:    cfg,
		engine: eng,
		ln:     ln,
		sem:    make(chan struct{}, cfg.MaxClients),
		stats:  stats.New(),
	}

	slog.Info("WarpDB listening", "port", cfg.Port, "max_clients", cfg.MaxClients)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("shutting down", "signal", sig.String())
		cancel()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				slog.Info("server stopped")
				return
			default:
				slog.Error("accept error", "error", err)
				continue
			}
		}
		select {
		case srv.sem <- struct{}{}:
			go srv.handleClient(ctx, conn)
		case <-ctx.Done():
			conn.Close()
			return
		default:
			protocol.WriteError(conn, "ERR max number of clients reached")
			conn.Close()
		}
	}
}

func (srv *Server) handleClient(ctx context.Context, conn net.Conn) {
	defer func() {
		conn.Close()
		<-srv.sem
	}()

	remoteAddr := conn.RemoteAddr().String()
	srv.stats.TotalConnections.Add(1)
	srv.stats.ActiveConnections.Add(1)
	defer srv.stats.ActiveConnections.Add(-1)
	slog.Debug("client connected", "addr", remoteAddr)

	reader := bufio.NewReader(conn)
	handler := commands.NewHandler(srv.engine, &srv.cfg, srv.stats)

	authenticated := srv.cfg.RequirePass == ""

	for {
		if srv.cfg.Timeout() > 0 {
			dl, _ := ctx.Deadline()
			conn.SetDeadline(dl)
		}

		args, err := protocol.Parse(reader)
		if err != nil {
			if ctx.Err() != nil {
				slog.Debug("client disconnected during shutdown", "addr", remoteAddr)
				return
			}
			slog.Debug("client disconnected", "addr", remoteAddr, "error", err)
			return
		}

		if !authenticated {
			if len(args) > 0 && strings.ToUpper(args[0]) == "AUTH" {
				if len(args) == 2 && args[1] == srv.cfg.RequirePass {
					authenticated = true
					protocol.WriteSimpleString(conn, "OK")
					slog.Debug("client authenticated", "addr", remoteAddr)
					continue
				}
				protocol.WriteError(conn, "ERR invalid password")
				continue
			}
			if !isUnauthenticatedCommand(args) {
				protocol.WriteError(conn, "NOAUTH Authentication required.")
				continue
			}
		}

		handler.Handle(args, conn)
	}
}

func isUnauthenticatedCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	cmd := strings.ToUpper(args[0])
	return cmd == "PING" || cmd == "QUIT" || cmd == "COMMAND"
}
