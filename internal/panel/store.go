package panel

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db    *sql.DB
	aead  cipher.AEAD
	key   []byte
	vault *blob.Vault
}

func Open(path string, key []byte) (*Store, error) {
	if len(key) != 32 || !filepath.IsAbs(path) {
		return nil, errors.New("absolute database path and 32-byte key required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("database directory must be private (0700)")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	f.Close()
	if err = os.Chmod(path, 0600); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	s := &Store{db: db, aead: aead, key: append([]byte(nil), key...)}
	success := false
	defer func() {
		if !success {
			db.Close()
		}
	}()
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA busy_timeout=5000", "PRAGMA foreign_keys=ON"} {
		if _, err = db.Exec(q); err != nil {
			return nil, err
		}
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return nil, err
	}
	if version > 4 {
		return nil, errors.New("database schema newer than application")
	}
	if version == 0 {
		tx, e := db.Begin()
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		_, err = tx.Exec(`
CREATE TABLE metadata (key TEXT PRIMARY KEY,value BLOB NOT NULL);
CREATE TABLE sources (id TEXT PRIMARY KEY,name TEXT NOT NULL,url BLOB NOT NULL,url_hash TEXT NOT NULL UNIQUE,display_url TEXT NOT NULL,min INTEGER NOT NULL,max INTEGER NOT NULL,unit TEXT NOT NULL,enabled INTEGER NOT NULL,revision INTEGER NOT NULL,created INTEGER NOT NULL,updated INTEGER NOT NULL,next_run INTEGER NOT NULL);
CREATE TABLE admin (id INTEGER PRIMARY KEY CHECK(id=1),password_hash TEXT NOT NULL);
CREATE TABLE sessions (hash TEXT PRIMARY KEY,expires INTEGER NOT NULL);
CREATE TABLE attempts (key TEXT PRIMARY KEY,count INTEGER NOT NULL,window INTEGER NOT NULL);
PRAGMA user_version=1;`)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	}
	// Reject the wrong master key immediately, even when there are no sources.
	var sentinel []byte
	err = db.QueryRow("SELECT value FROM metadata WHERE key='key-check'").Scan(&sentinel)
	if errors.Is(err, sql.ErrNoRows) {
		sentinel = s.seal("metadata", []byte("config-panel-key-v1"))
		_, err = db.Exec("INSERT INTO metadata VALUES ('key-check',?)", sentinel)
	} else if err == nil {
		var plain []byte
		plain, err = s.open("metadata", sentinel)
		if err == nil && string(plain) != "config-panel-key-v1" {
			err = errors.New("master key mismatch")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("database key verification failed: %w", err)
	}
	if err = s.migrateV2(); err != nil {
		return nil, err
	}
	if err = s.migrateV3(); err != nil {
		return nil, err
	}
	if err = s.migrateV4(); err != nil {
		return nil, err
	}
	s.vault, err = blob.New(filepath.Join(filepath.Dir(path), "blobs"), key)
	if err != nil {
		return nil, err
	}
	success = true
	return s, nil
}
func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Store) digest(scope, value string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(scope + "\x00" + value))
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Store) seal(id string, plain []byte) []byte {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}
	return s.aead.Seal(nonce, nonce, plain, []byte(id))
}
func (s *Store) open(id string, data []byte) ([]byte, error) {
	n := s.aead.NonceSize()
	if len(data) < n {
		return nil, errors.New("invalid ciphertext")
	}
	return s.aead.Open(nil, data[:n], data[n:], []byte(id))
}
