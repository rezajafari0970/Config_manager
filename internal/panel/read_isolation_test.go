package panel

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestReadDriverSeesCommittedStateWithoutWaitingForWriter(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	before, e := s.Settings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE settings SET concurrency=3"); e != nil {
		t.Fatal(e)
	}
	got, e := s.Settings(ctx)
	if e != nil || got != before {
		t.Fatalf("reader blocked or saw uncommitted state: %v %v", got, e)
	}
	if _, e = s.Stats(ctx); e != nil {
		t.Fatal(e)
	}
	for _, p := range []*SmartPipeline{s.Smart(), s.SplashSmart()} {
		if _, e = p.SmartSettings(ctx); e != nil {
			t.Fatal(e)
		}
		if _, e = p.SmartWorker(ctx); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	got, e = s.Settings(ctx)
	if e != nil || got.Concurrency != 3 {
		t.Fatal("reader missed committed update", e)
	}
}

func TestReadDriverIsBoundedReadOnlyAfterReconnect(t *testing.T) {
	s := testStore(t)
	if s.readDB.Stats().MaxOpenConnections != 3 || s.db.Stats().MaxOpenConnections != 1 {
		t.Fatal("unbounded reader or concurrent writers")
	}
	// Force physical connection replacement; DSN pragmas must survive it.
	s.readDB.SetMaxIdleConns(0)
	for i := 0; i < 2; i++ {
		c, e := s.readDB.Conn(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		var foreign, queryOnly int
		if e = c.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreign); e != nil {
			t.Fatal(e)
		}
		if e = c.QueryRowContext(context.Background(), "PRAGMA query_only").Scan(&queryOnly); e != nil {
			t.Fatal(e)
		}
		if foreign != 1 || queryOnly != 1 {
			t.Fatal("read connection pragmas lost")
		}
		if _, e = c.ExecContext(context.Background(), "UPDATE settings SET concurrency=3"); e == nil {
			t.Fatal("reader accepted mutation")
		}
		c.Close()
	}
}

func TestThinSettingsEndpointsAreIndependentOfPublicationAndWriter(t *testing.T) {
	s := testStore(t)
	app, e := NewServer(s, "https://panel.example")
	if e != nil {
		t.Fatal(e)
	}
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec("UPDATE smart_settings SET member_count=4"); e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	for _, path := range []string{"/api/settings", "/api/conversion/settings", "/api/smart/settings", "/api/smart-splash/settings"} {
		done := make(chan bool, 1)
		go func() {
			r := exportRequest(t, h, "GET", app.origin+path)
			var v map[string]json.RawMessage
			if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &v) != nil {
				t.Errorf("%s: status %d", path, r.Code)
			}
			if v["settings"] == nil || v["entries"] != nil || v["stats"] != nil {
				t.Errorf("settings endpoint contains listing work: %s", path)
			}
			done <- true
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("settings blocked behind unrelated writer", path)
		}
	}
}
