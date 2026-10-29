// Package blob stores immutable response/config bytes in encrypted streaming files.
package blob

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const chunkSize = 64 * 1024

type Ref struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}
type Vault struct {
	dir  string
	aead cipher.AEAD
	mu   sync.Mutex
	pins map[string]int
}

func New(dir string, key []byte) (*Vault, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid vault key")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	b, _ := aes.NewCipher(key)
	a, _ := cipher.NewGCM(b)
	return &Vault{dir: dir, aead: a, pins: map[string]int{}}, nil
}
func (v *Vault) Directory() string { return v.dir }
func nonce(base []byte, n uint64) []byte {
	b := append([]byte(nil), base...)
	value := binary.BigEndian.Uint64(b[4:])
	binary.BigEndian.PutUint64(b[4:], value^n)
	return b
}
func validHash(h string) bool         { b, e := hex.DecodeString(h); return e == nil && len(b) == 32 }
func (v *Vault) path(h string) string { return filepath.Join(v.dir, h+".blob") }

// Put pins the resulting file until Release. A DB publication must finish before Release.
func (v *Vault) Put(ctx context.Context, r io.Reader) (Ref, error) {
	f, err := os.CreateTemp(v.dir, ".write-")
	if err != nil {
		return Ref{}, err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	header := make([]byte, 16)
	copy(header, "CRV1")
	if _, err = rand.Read(header[4:]); err != nil {
		return Ref{}, err
	}
	if _, err = f.Write(header); err != nil {
		return Ref{}, err
	}
	hash := sha256.New()
	buf := make([]byte, chunkSize)
	var size int64
	var sequence uint64
	for {
		if err = ctx.Err(); err != nil {
			return Ref{}, err
		}
		n, readErr := io.ReadFull(r, buf)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return Ref{}, readErr
		}
		if n > 0 {
			hash.Write(buf[:n])
			size += int64(n)
			sealed := v.aead.Seal(nil, nonce(header[4:], sequence), buf[:n], header)
			sequence++
			var length [4]byte
			binary.BigEndian.PutUint32(length[:], uint32(len(sealed)))
			if _, err = f.Write(length[:]); err != nil {
				return Ref{}, err
			}
			if _, err = f.Write(sealed); err != nil {
				return Ref{}, err
			}
		}
		if readErr != nil {
			break
		}
	}
	terminal := v.aead.Seal(nil, nonce(header[4:], sequence), nil, header)
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(terminal)))
	if _, err = f.Write(length[:]); err != nil {
		return Ref{}, err
	}
	if _, err = f.Write(terminal); err != nil {
		return Ref{}, err
	}
	if err = f.Sync(); err != nil {
		return Ref{}, err
	}
	if err = f.Close(); err != nil {
		return Ref{}, err
	}
	ref := Ref{Hash: hex.EncodeToString(hash.Sum(nil)), Size: size}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err = os.Link(name, v.path(ref.Hash)); err != nil && !os.IsExist(err) {
		return Ref{}, err
	}
	v.pins[ref.Hash]++
	if d, e := os.Open(v.dir); e == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return ref, nil
}
func (v *Vault) Release(hash string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.pins[hash]--
	if v.pins[hash] <= 0 {
		delete(v.pins, hash)
	}
}
func (v *Vault) Open(hash string) (io.ReadCloser, error) {
	if !validHash(hash) {
		return nil, os.ErrNotExist
	}
	v.mu.Lock()
	f, err := os.Open(v.path(hash))
	v.mu.Unlock()
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	go func() { defer f.Close(); err := v.decode(f, w, hash); _ = w.CloseWithError(err) }()
	return r, nil
}
func (v *Vault) decode(r io.Reader, w io.Writer, expected string) error {
	header := make([]byte, 16)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	if string(header[:4]) != "CRV1" {
		return errors.New("invalid blob")
	}
	hash := sha256.New()
	var sequence uint64
	for {
		var length [4]byte
		if _, err := io.ReadFull(r, length[:]); err != nil {
			return err
		}
		size := binary.BigEndian.Uint32(length[:])
		if size < uint32(v.aead.Overhead()) || size > chunkSize+uint32(v.aead.Overhead()) {
			return errors.New("invalid blob chunk")
		}
		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return err
		}
		plain, err := v.aead.Open(nil, nonce(header[4:], sequence), data, header)
		sequence++
		if err != nil {
			return err
		}
		if len(plain) == 0 {
			var extra [1]byte
			n, e := r.Read(extra[:])
			if n != 0 || e != io.EOF {
				return errors.New("trailing blob data")
			}
			if hex.EncodeToString(hash.Sum(nil)) != expected {
				return errors.New("blob checksum mismatch")
			}
			return nil
		}
		hash.Write(plain)
		if _, err = w.Write(plain); err != nil {
			return err
		}
	}
}

// Sweep checks database references while holding the same lock as publication pins.
func (v *Vault) Sweep(ctx context.Context, referenced func(context.Context, string) (bool, error)) error {
	entries, err := os.ReadDir(v.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".write-") {
			if info, e := entry.Info(); e == nil && time.Since(info.ModTime()) > time.Hour {
				_ = os.Remove(filepath.Join(v.dir, name))
			}
			continue
		}
		hash := strings.TrimSuffix(name, ".blob")
		if !validHash(hash) {
			continue
		}
		v.mu.Lock()
		if v.pins[hash] > 0 {
			v.mu.Unlock()
			continue
		}
		used, e := referenced(ctx, hash)
		if e == nil && !used {
			e = os.Remove(v.path(hash))
			if os.IsNotExist(e) {
				e = nil
			}
		}
		v.mu.Unlock()
		if e != nil {
			return e
		}
	}
	return nil
}
