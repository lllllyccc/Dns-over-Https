package server

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/miekg/dns"

	"dns-over-https/internal/logger"
	"dns-over-https/internal/resolver"
)

type DoHHandler struct {
	fwd    *resolver.Forwarder
	logger *logger.Logger
}

func NewDoHHandler(fwd *resolver.Forwarder, log *logger.Logger) *DoHHandler {
	return &DoHHandler{
		fwd:    fwd,
		logger: log,
	}
}

func (h *DoHHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawMsg []byte
	var err error

	switch r.Method {
	case http.MethodGet:
		dnsParam := r.URL.Query().Get("dns")
		if dnsParam == "" {
			http.Error(w, "missing dns parameter", http.StatusBadRequest)
			return
		}
		rawMsg, err = base64.RawURLEncoding.DecodeString(dnsParam)
		if err != nil {
			http.Error(w, "invalid dns parameter", http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		ct := r.Header.Get("Content-Type")
		if ct != "application/dns-message" {
			http.Error(w, "invalid content type", http.StatusUnsupportedMediaType)
			return
		}
		rawMsg, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	msg := new(dns.Msg)
	if err := msg.Unpack(rawMsg); err != nil {
		http.Error(w, "invalid dns message", http.StatusBadRequest)
		return
	}

	start := time.Now()
	resp, upstream, err := h.fwd.Resolve(r.Context(), msg)
	rtt := time.Since(start).Milliseconds()

	clientIP := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		clientIP = fwd
	}

	qName := ""
	qType := ""
	blocked := false
	if len(msg.Question) > 0 {
		qName = msg.Question[0].Name
		qType = dns.TypeToString[msg.Question[0].Qtype]
	}
	if upstream == "filter" {
		blocked = true
	}

	h.logger.LogQuery(logger.QueryEntry{
		Timestamp: time.Now(),
		ClientIP:  clientIP,
		QueryName: qName,
		QueryType: qType,
		Upstream:  upstream,
		CacheHit:  upstream == "cache",
		Blocked:   blocked,
		RTT:       rtt,
	})

	if err != nil {
		h.logger.LogError("resolve failed", "error", err)
		http.Error(w, "resolution failed", http.StatusInternalServerError)
		return
	}

	respData, err := resp.Pack()
	if err != nil {
		h.logger.LogError("pack response failed", "error", err)
		http.Error(w, "failed to pack response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/dns-message")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Write(respData)
}

var _ = context.Background
