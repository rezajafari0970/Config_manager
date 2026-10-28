package panel

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

var ErrAuth = errors.New("authentication failed")

func passwordHash(password string) (string, error) {
	if len(password) < 14 || len(password) > 1024 {
		return "", errors.New("password must contain 14–1024 bytes")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return "argon2id-v1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}
func passwordMatches(encoded, password string) bool {
	if len(password) > 1024 {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "argon2id-v1" {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[1])
	expected, e2 := base64.RawStdEncoding.DecodeString(parts[2])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(expected) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
func (s *Store) InitAdmin(ctx context.Context, password string) error {
	hash, err := passwordHash(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO admin (id,password_hash) VALUES (1,?)", hash)
	return err
}
func (s *Store) HasAdmin(ctx context.Context) bool {
	var n int
	return s.db.QueryRowContext(ctx, "SELECT count(*) FROM admin").Scan(&n) == nil && n == 1
}
func sessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *Store) Authenticate(ctx context.Context, password string, remember bool) (string, time.Time, error) {
	var hash string
	if err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM admin WHERE id=1").Scan(&hash); err != nil {
		return "", time.Time{}, err
	}
	if !passwordMatches(hash, password) {
		return "", time.Time{}, ErrAuth
	}
	ttl := 2 * time.Hour
	if remember {
		ttl = 30 * 24 * time.Hour
	}
	token := randomToken()
	expires := time.Now().Add(ttl)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback()
	var currentHash string
	if err = tx.QueryRowContext(ctx, "SELECT password_hash FROM admin WHERE id=1").Scan(&currentHash); err != nil {
		return "", time.Time{}, err
	}
	if currentHash != hash {
		return "", time.Time{}, ErrAuth
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE expires<=?", time.Now().Unix()); err != nil {
		return "", time.Time{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO sessions VALUES (?,?)", sessionHash(token), expires.Unix()); err != nil {
		return "", time.Time{}, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE hash IN (SELECT hash FROM sessions ORDER BY expires DESC LIMIT -1 OFFSET 100)"); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, tx.Commit()
}
func (s *Store) Session(ctx context.Context, token string) bool {
	if len(token) != 43 {
		return false
	}
	var expires int64
	return s.db.QueryRowContext(ctx, "SELECT expires FROM sessions WHERE hash=?", sessionHash(token)).Scan(&expires) == nil && time.Now().Unix() < expires
}
func (s *Store) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE hash=?", sessionHash(token))
	return err
}
func (s *Store) ChangePassword(ctx context.Context, current, next string) error {
	var old string
	if err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM admin WHERE id=1").Scan(&old); err != nil {
		return err
	}
	if !passwordMatches(old, current) {
		return ErrAuth
	}
	hash, err := passwordHash(next)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE admin SET password_hash=? WHERE id=1 AND password_hash=?", hash, old)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions"); err != nil {
		return err
	}
	return tx.Commit()
}

// Reserve before hashing; both per-client and global limits survive restarts.
func (s *Store) AllowAuth(ctx context.Context, client string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	cutoff := now - 15*60
	if _, err = tx.ExecContext(ctx, "DELETE FROM attempts WHERE window<=?", cutoff); err != nil {
		return false, err
	}
	for _, item := range []struct {
		key   string
		limit int
	}{{s.digest("auth-ip", client), 10}, {"global", 100}} {
		var n int
		err = tx.QueryRowContext(ctx, "SELECT count FROM attempts WHERE key=?", item.key).Scan(&n)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
		if n >= item.limit {
			return false, nil
		}
	}
	for _, key := range []string{s.digest("auth-ip", client), "global"} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO attempts VALUES (?,1,?) ON CONFLICT(key) DO UPDATE SET count=count+1", key, now); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}
