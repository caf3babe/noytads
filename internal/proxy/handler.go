package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/elazarl/goproxy"
)

// Metrics tracks proxy statistics.
var Metrics struct {
	Passed   atomic.Int64
	Modified atomic.Int64
	Blocked  atomic.Int64
}

// RegisterHandlers sets up response modification for target domains.
func RegisterHandlers(p *goproxy.ProxyHttpServer) {
	p.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil || resp.Body == nil {
			return resp
		}

		host := ctx.Req.URL.Host
		contentType := resp.Header.Get("Content-Type")

		// Phase 1: only modify HTML responses from target domains
		if !isTargetDomain(host) || !strings.Contains(contentType, "text/html") {
			Metrics.Passed.Add(1)
			return resp
		}

		body, err := readBody(resp)
		if err != nil {
			log.Printf("[handler] error reading body from %s: %v", host, err)
			return resp
		}

		modified := InjectIntoHTML(body, host)
		if modified == nil {
			Metrics.Passed.Add(1)
			return resp
		}

		// Replace response body with modified content
		resp.Body = io.NopCloser(bytes.NewReader(modified))
		resp.ContentLength = int64(len(modified))
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modified)))
		resp.Header.Del("Content-Encoding") // We decompressed, send plain

		Metrics.Modified.Add(1)
		if p.Verbose {
			log.Printf("[handler] injected into %s (%d bytes → %d bytes)",
				host, len(body), len(modified))
		}

		return resp
	})
}

// isTargetDomain checks if the host should be intercepted.
// Phase 1: hardcoded YouTube domains.
// Phase 2: this will be driven by the rule engine index.
func isTargetDomain(host string) bool {
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	return strings.HasSuffix(host, "youtube.com") ||
		strings.HasSuffix(host, "youtubekids.com") ||
		strings.HasSuffix(host, "youtube-nocookie.com")
}

// readBody handles gzip-compressed and plain responses.
func readBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	defer resp.Body.Close()

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	case "br":
		// TODO Phase 2: brotli decompression
		// For now, skip brotli-compressed responses
		return nil, fmt.Errorf("brotli not yet supported")
	}

	// Limit to 5 MB to avoid OOM on router hardware
	return io.ReadAll(io.LimitReader(reader, 5*1024*1024))
}
