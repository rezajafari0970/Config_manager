package collect

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublicDialSkipsBlackholeAndCancelsLoser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ips := []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1")}
	firstStarted, firstStopped := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	peer, server := net.Pipe()
	defer server.Close()
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		calls.Add(1)
		if address == "8.8.8.8:443" {
			close(firstStarted)
			<-ctx.Done()
			close(firstStopped)
			return nil, ctx.Err()
		}
		<-firstStarted
		return peer, nil
	}
	c, e := dialPublic(ctx, "tcp", "443", ips, dial, time.Millisecond, 200*time.Millisecond)
	if e != nil || c != peer || calls.Load() != 2 {
		t.Fatal("healthy fallback not selected", e)
	}
	c.Close()
	select {
	case <-firstStopped:
	case <-ctx.Done():
		t.Fatal("loser not canceled")
	}
}

func TestPublicDialBlocksMixedDNSBeforeAnyConnection(t *testing.T) {
	called := false
	_, e := dialPublic(context.Background(), "tcp", "443", []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}, func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, errors.New("unexpected")
	}, time.Millisecond, time.Second)
	if !errors.Is(e, ErrBlocked) || called {
		t.Fatal("mixed private answer bypassed destination gate")
	}
}

func TestPublicDialCapsConcurrencyAndReachesLaterAddresses(t *testing.T) {
	ips := []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.4.4")}
	var active, peak atomic.Int32
	dial := func(ctx context.Context, _, _ string) (net.Conn, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	_, e := dialPublic(context.Background(), "tcp", "443", ips, dial, time.Millisecond, 10*time.Millisecond)
	if e == nil || peak.Load() > 2 || active.Load() != 0 {
		t.Fatal("dial bounds", e, peak.Load(), active.Load())
	}
}
