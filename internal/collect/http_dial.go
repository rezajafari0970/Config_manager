package collect

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"time"
)

// Resolve once, validate every address, then dial literals only. A blackholed
// first address cannot consume the complete source deadline. At most two dials
// run concurrently; each has a bounded deadline and losing connections close.
func dialPublic(ctx context.Context, network, port string, ips []netip.Addr, dial func(context.Context, string, string) (net.Conn, error), stagger, attemptTimeout time.Duration) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, errors.New("dns_empty")
	}
	for _, ip := range ips {
		if !PublicIP(ip) {
			return nil, ErrBlocked
		}
	}
	// Alternate address families while preserving resolver preference in each.
	first, other := []netip.Addr{}, []netip.Addr{}
	for _, ip := range ips {
		if ip.Is4() == ips[0].Is4() {
			first = append(first, ip)
		} else {
			other = append(other, ip)
		}
	}
	order := make([]netip.Addr, 0, len(ips))
	for i := 0; i < len(first) || i < len(other); i++ {
		if i < len(first) {
			order = append(order, first[i])
		}
		if i < len(other) {
			order = append(order, other[i])
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		conn net.Conn
		err  error
	}
	results := make(chan result)
	next, active := 0, 0
	start := func() {
		ip := order[next]
		next++
		active++
		go func() {
			attempt, stop := context.WithTimeout(ctx, attemptTimeout)
			defer stop()
			c, e := dial(attempt, network, net.JoinHostPort(ip.Unmap().String(), port))
			if e != nil && c != nil {
				c.Close()
				c = nil
			}
			select {
			case results <- result{c, e}:
			case <-ctx.Done():
				if c != nil {
					c.Close()
				}
			}
		}()
	}
	start()
	timer := time.NewTimer(stagger)
	defer timer.Stop()
	var last error
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			if active < 2 && next < len(order) {
				start()
			}
		case r := <-results:
			active--
			if r.err == nil && r.conn != nil {
				if ctx.Err() != nil {
					r.conn.Close()
					return nil, ctx.Err()
				}
				return r.conn, nil
			}
			last = r.err
			if next < len(order) {
				start()
			}
			if active == 0 {
				if last == nil {
					last = errors.New("dial_failed")
				}
				return nil, last
			}
		}
	}
}
