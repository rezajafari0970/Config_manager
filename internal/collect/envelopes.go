package collect

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// Envelopes are transport encodings, not repairs. Keep the response blob and
// extract exact decoded payloads without serializing JSON or splitting outbounds.
// This bounds nesting work, not the response size or number of configurations.
const maxEnvelopeDepth = 8

func envelopeEnd(ctx context.Context, data []byte, start, end int) int {
	last := bytes.TrimSpace(data[start:end])
	for end < len(data) {
		if ctx.Err() != nil || bytes.HasSuffix(last, []byte("=")) {
			break
		}
		next := bytes.IndexByte(data[end:], '\n')
		if next < 0 {
			next = len(data)
		} else {
			next += end + 1
		}
		line := bytes.TrimSpace(data[end:next])
		if len(line) == 0 {
			end = next
			continue
		}
		if !base64Alphabet(line) {
			break
		}
		if envelopeStartsConfig(line, 0) && !unfinishedEncodedJSON(ctx, data[start:end]) {
			break
		}
		// A continuation line does not independently start a config. Decode
		// the entire wrapped block once, never emit a truncated first line.
		last, end = line, next
	}
	return end
}

func envelopeStartsConfig(b []byte, depth int) bool {
	if depth >= maxEnvelopeDepth {
		return false
	}
	// Only the prefix is needed to distinguish an independent encoded record
	// from a wrapped continuation. The full decoder below validates the block.
	prefix := make([]byte, 0, 512)
	for _, c := range b {
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		if c == '=' || len(prefix) == cap(prefix) {
			break
		}
		prefix = append(prefix, c)
	}
	prefix = prefix[:len(prefix)/4*4]
	encoding := base64.RawStdEncoding
	if bytes.ContainsAny(prefix, "-_") {
		encoding = base64.RawURLEncoding
	}
	decoded, err := encoding.DecodeString(string(prefix))
	if err != nil {
		return false
	}
	decoded = bytes.TrimSpace(bytes.TrimPrefix(decoded, []byte{0xef, 0xbb, 0xbf}))
	if len(decoded) == 0 {
		return false
	}
	if kindOf(decoded) != "" || decoded[0] == '{' || decoded[0] == '[' || decoded[0] == '"' {
		return true
	}
	return base64Prefix(decoded) && envelopeStartsConfig(decoded, depth+1)
}

func (d textFormat) emitBase64(ctx context.Context, raw []byte, emit Emitter) error {
	if d.depth >= maxEnvelopeDepth {
		return errors.New("envelope_nesting_limit")
	}
	trim := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
	encoding := base64.RawStdEncoding
	if bytes.ContainsAny(trim, "-_") {
		encoding = base64.RawURLEncoding
	}
	if bytes.HasSuffix(trim, []byte("=")) {
		encoding = encoding.WithPadding(base64.StdPadding)
	}
	out, err := os.CreateTemp("", "config-envelope-")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	defer out.Close()
	reader := &envelopeWhitespaceReader{r: bufio.NewReader(&contextReader{ctx, bytes.NewReader(trim)})}
	_, err = io.Copy(out, base64.NewDecoder(encoding, reader))
	if err != nil {
		var corrupt base64.CorruptInputError
		if errors.As(err, &corrupt) || errors.Is(err, io.ErrUnexpectedEOF) {
			if envelopeStartsConfig(trim, 0) {
				return errors.New("invalid_base64_envelope")
			}
			return nil // ordinary text need not be a base64 envelope
		}
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	registry := d.registry
	if registry == nil {
		registry = NewRegistry()
	}
	return registry.extract(ctx, out.Name(), emit, d.depth+1)
}

// Unlike stripping all '=' characters, this preserves padding for the decoder
// to reject internal padding/truncated blocks rather than corrupt credentials.
type envelopeWhitespaceReader struct{ r *bufio.Reader }

func (r *envelopeWhitespaceReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		b, err := r.r.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if b == ' ' || b == '\r' || b == '\n' || b == '\t' {
			continue
		}
		p[n] = b
		n++
		if r.r.Buffered() == 0 {
			break
		}
	}
	return n, nil
}

// A wrapped JSON continuation can itself start with a complete child object.
// Do not treat that child as a new encoded record while its parent is open.
// Scan decoded bytes in fixed buffers; no JSON value/string is materialized.
func unfinishedEncodedJSON(ctx context.Context, raw []byte) bool {
	trim := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
	encoding := base64.RawStdEncoding
	if bytes.ContainsAny(trim, "-_") {
		encoding = base64.RawURLEncoding
	}
	if bytes.HasSuffix(trim, []byte("=")) {
		encoding = encoding.WithPadding(base64.StdPadding)
	}
	r := base64.NewDecoder(encoding, &envelopeWhitespaceReader{r: bufio.NewReader(&contextReader{ctx, bytes.NewReader(trim)})})
	var buffer [4096]byte
	depth := 0
	started, quoted, escaped := false, false, false
	for {
		n, err := r.Read(buffer[:])
		for _, c := range buffer[:n] {
			if !started {
				if c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == 0xef || c == 0xbb || c == 0xbf {
					continue
				}
				if c != '{' && c != '[' && c != '"' {
					return false
				}
				started = true
			}
			if quoted {
				if escaped {
					escaped = false
				} else if c == '\\' {
					escaped = true
				} else if c == '"' {
					quoted = false
					if depth == 0 {
						started = false
					}
				}
				continue
			}
			switch c {
			case '"':
				quoted = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					started = false
				}
			}
		}
		if err != nil {
			return started
		}
	}
}
