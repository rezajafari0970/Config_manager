// Synthetic HTTP/2 DoH fixture; built only by smart-bootstrap-test.py.
package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	work := os.Args[1]
	var mu sync.Mutex
	handler := func(service string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var b []byte
			var e error
			if r.Method == http.MethodGet {
				b, e = base64.RawURLEncoding.DecodeString(r.URL.Query().Get("dns"))
			} else {
				b, e = io.ReadAll(io.LimitReader(r.Body, 4096))
			}
			if e != nil || len(b) < 17 {
				http.Error(w, "invalid query", 400)
				return
			}
			end := 12
			labels := []string{}
			for end < len(b) && b[end] != 0 {
				n := int(b[end])
				if n > 63 || end+1+n >= len(b) {
					http.Error(w, "invalid query", 400)
					return
				}
				labels = append(labels, string(b[end+1:end+1+n]))
				end += 1 + n
			}
			end++
			if end+4 > len(b) {
				http.Error(w, "invalid question", 400)
				return
			}
			name := strings.Join(labels, ".")
			kind := binary.BigEndian.Uint16(b[end : end+2])
			peer, _, _ := net.SplitHostPort(r.RemoteAddr)
			allowed := peer == "127.0.0.10" || (name != "peer.test" && peer == "127.0.0.11")
			item, _ := json.Marshal(map[string]any{"name": name, "kind": kind, "peer": peer, "service": service, "allowed": allowed, "http": r.ProtoMajor, "at": time.Now().UnixNano()})
			mu.Lock()
			f, e := os.OpenFile(filepath.Join(work, "trace.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if e != nil {
				panic(e)
			}
			f.Write(append(item, '\n'))
			f.Close()
			mu.Unlock()
			if !allowed {
				http.Error(w, "direct or circular DNS", 403)
				return
			}
			time.Sleep(80 * time.Millisecond)
			data := []byte{2, 'o', 'k'}
			if kind == 1 {
				data = []byte{203, 0, 113, 7}
				if name == "peer.test" {
					data = []byte{127, 0, 0, 1}
				}
			}
			reply := append([]byte{}, b[:2]...)
			reply = append(reply, 0x81, 0x80, 0, 1, 0, 1, 0, 0, 0, 0)
			reply = append(reply, b[12:end+4]...)
			reply = append(reply, 0xc0, 0x0c, byte(kind>>8), byte(kind), 0, 1, 0, 0, 0, 30, 0, byte(len(data)))
			reply = append(reply, data...)
			w.Header().Set("Content-Type", "application/dns-message")
			w.Write(reply)
		}
	}
	for _, ip := range []string{"127.0.0.2", "127.0.0.3"} {
		listener, e := net.Listen("tcp", ip+":24454")
		if e != nil {
			panic(e)
		}
		server := &http.Server{Handler: handler(ip), ReadHeaderTimeout: 3 * time.Second}
		go func() {
			if e := server.ServeTLS(listener, filepath.Join(work, "cert.pem"), filepath.Join(work, "key.pem")); e != nil {
				fmt.Fprintln(os.Stderr, e)
				os.Exit(1)
			}
		}()
	}
	if e := os.WriteFile(filepath.Join(work, "doh-ready"), []byte("ready"), 0600); e != nil {
		panic(e)
	}
	select {}
}
