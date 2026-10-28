package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/panel"
)

type config struct {
	Listen       string `json:"listen"`
	PublicOrigin string `json:"public_origin"`
	Database     string `json:"database"`
	MasterKey    string `json:"master_key"`
}

func run() error {
	file := flag.String("config", "/etc/config-repair/panel.json", "protected runtime configuration")
	init := flag.Bool("init-admin", false, "initialize the only administrator, once")
	passwordFile := flag.String("password-file", "", "0600 file containing initial password")
	flag.Parse()
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0007 != 0 {
		return errors.New("config must not be accessible to other users")
	}
	var c config
	d := json.NewDecoder(io.LimitReader(f, 16384))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return errors.New("invalid runtime configuration")
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return errors.New("extra runtime configuration data")
	}
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return errors.New("listen must be a loopback IP:port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("only loopback listener supported; use TLS reverse proxy")
	}
	key, err := base64.StdEncoding.DecodeString(c.MasterKey)
	if err != nil {
		return errors.New("invalid master key")
	}
	s, err := panel.Open(c.Database, key)
	if err != nil {
		return err
	}
	defer s.Close()
	if *init {
		if *passwordFile == "" {
			return errors.New("password-file required")
		}
		info, err := os.Stat(*passwordFile)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0077 != 0 {
			return errors.New("password file must be 0600")
		}
		p, err := os.ReadFile(*passwordFile)
		if err != nil {
			return err
		}
		if err = s.InitAdmin(context.Background(), strings.TrimSuffix(string(p), "\n")); err != nil {
			return errors.New("admin initialization failed (already initialized or invalid password)")
		}
		fmt.Println("Administrator initialized")
		return nil
	}
	if !s.HasAdmin(context.Background()) {
		return errors.New("initialize administrator before starting server")
	}
	app, err := panel.NewServer(s, c.PublicOrigin)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: c.Listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
		case <-done:
		}
	}()
	log.Print("config panel listening on loopback; fetch worker disabled")
	err = server.ListenAndServe()
	close(done)
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
