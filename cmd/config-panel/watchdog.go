package main

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/panel"
	"net"
	"os"
	"time"
)

func watchdog(ctx context.Context, engine *collect.Engine, store *panel.Store) {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			running, _, tick := engine.Status()
			if !running || time.Now().UnixMilli()-tick > 20000 {
				continue
			}
			check, cancel := context.WithTimeout(ctx, 2*time.Second)
			e := store.Ping(check)
			cancel()
			if e != nil {
				continue
			}
			connection, e := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
			if e != nil {
				continue
			}
			_ = connection.SetWriteDeadline(time.Now().Add(time.Second))
			_, _ = connection.Write([]byte("WATCHDOG=1"))
			connection.Close()
		}
	}
}
