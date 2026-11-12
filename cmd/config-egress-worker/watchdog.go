package main

import (
	"net"
	"os"
	"strings"
	"time"
)

// The optional systemd driver is independent of queue/API availability. The
// coordinator emits watchdog ticks only while its own loop is making progress.
func notifySystemd(message string) {
	path := os.Getenv("NOTIFY_SOCKET")
	if path == "" {
		return
	}
	if strings.HasPrefix(path, "@") {
		path = "\x00" + path[1:]
	}
	c, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		return
	}
	defer c.Close()
	_ = c.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = c.Write([]byte(message))
}
