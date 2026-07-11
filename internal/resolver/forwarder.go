package resolver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/miekg/dns"

	"dns-over-https/internal/filter"
)

type Upstream struct {
	Name     string
	Address  string
	Protocol string
	Weight   int
	client   *dns.Client
	dohURL  string
}

type Forwarder struct {
	upstreams []*Upstream
	cache     *Cache
	filter    *filter.Filter
	mu        sync.RWMutex
}

func NewForwarder(upstreams []UpstreamConfig, cache *Cache, flt *filter.Filter) *Forwarder {
	f := &Forwarder{
		cache:  cache,
		filter: flt,
	}
	for _, u := range upstreams {
		f.upstreams = append(f.upstreams, &Upstream{
			Name:     u.Name,
			Address:  u.Address,
			Protocol: u.Protocol,
			Weight:   u.Weight,
			client: &dns.Client{
				Net:     u.Protocol,
				Timeout: 5 * time.Second,
			},
		})
	}
	return f
}

type UpstreamConfig struct {
	Name     string
	Address  string
	Protocol string
	Weight   int
}

func (f *Forwarder) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if f.filter != nil && f.filter.IsEnabled() && len(q.Question) > 0 {
		name := q.Question[0].Name
		if f.filter.IsBlocked(name) {
			resp := new(dns.Msg)
			resp.SetRcode(q, dns.RcodeNameError)
			return resp, "filter", nil
		}
	}

	if f.cache != nil {
		if resp, ok := f.cache.Get(q); ok {
			return resp, "cache", nil
		}
	}

	var lastErr error
	for _, u := range f.upstreams {
		resp, err := f.queryUpstream(ctx, u, q)
		if err != nil {
			lastErr = err
			continue
		}
		if resp != nil && resp.Rcode == dns.RcodeSuccess {
			if f.cache != nil {
				f.cache.Set(q, resp)
			}
			return resp, u.Name, nil
		}
	}

	return nil, "", fmt.Errorf("all upstreams failed, last error: %v", lastErr)
}

func (f *Forwarder) queryUpstream(ctx context.Context, u *Upstream, q *dns.Msg) (*dns.Msg, error) {
	switch u.Protocol {
	case "doh":
		return f.queryDoH(ctx, u, q)
	default:
		return f.queryDNS(ctx, u, q)
	}
}

func (f *Forwarder) queryDNS(ctx context.Context, u *Upstream, q *dns.Msg) (*dns.Msg, error) {
	r, rtt, err := u.client.ExchangeContext(ctx, q, u.Address)
	if err != nil {
		return nil, fmt.Errorf("upstream %s: %w (rtt: %v)", u.Name, err, rtt)
	}
	return r, nil
}

func (f *Forwarder) queryDoH(ctx context.Context, u *Upstream, q *dns.Msg) (*dns.Msg, error) {
	data, err := q.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.Address, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doh request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	msg := new(dns.Msg)
	if err := msg.Unpack(respData); err != nil {
		return nil, fmt.Errorf("unpack response: %w", err)
	}

	return msg, nil
}

func (f *Forwarder) HealthCheck() map[string]bool {
	results := make(map[string]bool)
	for _, u := range f.upstreams {
		if u.Protocol == "doh" {
			results[u.Name] = true
			continue
		}
		conn, err := net.DialTimeout("udp", u.Address, 2*time.Second)
		if err != nil {
			results[u.Name] = false
		} else {
			conn.Close()
			results[u.Name] = true
		}
	}
	return results
}

func (f *Forwarder) UpstreamNames() []string {
	var names []string
	for _, u := range f.upstreams {
		names = append(names, u.Name)
	}
	return names
}
