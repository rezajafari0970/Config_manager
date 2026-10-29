package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"github.com/rezajafari0970/Config_manager/internal/panel"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type config struct {
	Listen       string `json:"listen"`
	PublicOrigin string `json:"public_origin"`
	Database     string `json:"database"`
	MasterKey    string `json:"master_key"`
}

func run() error {
	file := flag.String("config", "/etc/config-repair/panel.json", "protected runtime configuration")
	flag.Parse()
	f, e := os.Open(*file)
	if e != nil {
		return e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return e
	}
	if info.Mode().Perm()&0007 != 0 {
		return errors.New("runtime config must be private")
	}
	var c config
	d := json.NewDecoder(io.LimitReader(f, 16384))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil {
		return errors.New("invalid runtime configuration")
	}
	if e = d.Decode(new(any)); e != io.EOF {
		return errors.New("extra runtime configuration data")
	}
	host, _, e := net.SplitHostPort(c.Listen)
	if e != nil {
		return errors.New("loopback listener required")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("loopback listener required")
	}
	key, e := base64.StdEncoding.DecodeString(c.MasterKey)
	if e != nil {
		return errors.New("invalid master key")
	}
	if e = os.MkdirAll(filepath.Dir(c.Database), 0700); e != nil {
		return e
	}
	lock, e := os.OpenFile(filepath.Join(filepath.Dir(c.Database), "worker.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return errors.New("another collector owns this database")
	}
	store, e := panel.Open(c.Database, key)
	if e != nil {
		return e
	}
	defer store.Close()
	app, e := panel.NewServer(store, c.PublicOrigin)
	if e != nil {
		return e
	}
	socketPath := filepath.Join(filepath.Dir(c.Database), "probe.sock")
	if e = os.Remove(socketPath); e != nil && !os.IsNotExist(e) {
		return e
	}
	listener, e := net.Listen("unix", socketPath)
	if e != nil {
		return e
	}
	defer listener.Close()
	defer os.Remove(socketPath)
	if e = os.Chmod(socketPath, 0600); e != nil {
		return e
	}
	privateServer := &http.Server{Handler: store.PrivateProbeHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	privateResult := make(chan error, 1)
	go func() { privateResult <- privateServer.Serve(listener) }()
	defer privateServer.Close()
	engine := store.NewEngine()
	app.SetEngine(engine)
	server := &http.Server{Addr: c.Listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	engineResult := make(chan error, 1)
	go func() { engineResult <- engine.Run(ctx) }()
	go watchdog(ctx, engine, store)
	httpResult := make(chan error, 1)
	go func() { httpResult <- server.ListenAndServe() }()
	log.Print("config manager running; passwordless management and raw collector enabled")
	select {
	case <-ctx.Done():
	case e = <-privateResult:
		stop()
	case e = <-engineResult:
		if e == nil {
			e = errors.New("collector stopped unexpectedly")
		}
		stop()
		engineResult = nil
	case e = <-httpResult:
		if errors.Is(e, http.ErrServerClosed) {
			e = nil
		}
		stop()
		httpResult = nil
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	if engineResult != nil {
		if workerErr := <-engineResult; e == nil {
			e = workerErr
		}
	}
	return e
}
func main() {
	if e := run(); e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
