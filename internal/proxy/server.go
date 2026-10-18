package proxy

import (
	"fmt"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
)

// Config holds all proxy configuration.
type Config struct {
	ListenAddr   string
	CAPath       string
	CAKeyPath    string
	CAExportPort string
	LogLevel     string
}

// Server is the main proxy server.
type Server struct {
	config *Config
	proxy  *goproxy.ProxyHttpServer
	http   *http.Server
}

// NewServer creates and configures a new proxy server.
func NewServer(cfg *Config) (*Server, error) {
	// Ensure CA exists (generate on first run)
	ca, err := LoadOrGenerateCA(cfg.CAPath, cfg.CAKeyPath)
	if err != nil {
		return nil, fmt.Errorf("CA setup: %w", err)
	}
	log.Printf("CA loaded from %s", cfg.CAPath)

	// Setup goproxy with MITM
	p := goproxy.NewProxyHttpServer()
	p.Verbose = cfg.LogLevel == "debug"

	// Configure MITM for all HTTPS connections
	if err := ConfigureMITM(p, ca); err != nil {
		return nil, fmt.Errorf("MITM setup: %w", err)
	}

	// Register response handlers (injection pipeline)
	RegisterHandlers(p)

	srv := &Server{
		config: cfg,
		proxy:  p,
		http: &http.Server{
			Addr:    cfg.ListenAddr,
			Handler: p,
		},
	}

	// Start CA export server so iOS devices can download the cert
	go serveCAExport(cfg.CAExportPort, cfg.CAPath)

	return srv, nil
}

// ListenAndServe starts the proxy.
func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

// Shutdown stops the proxy gracefully.
func (s *Server) Shutdown() {
	s.http.Close()
}

// serveCAExport provides a simple web page for installing the CA on iOS.
func serveCAExport(addr, caPath string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ca.pem", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", "attachment; filename=noytads-ca.pem")
		http.ServeFile(w, r, caPath)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>noytads — Install CA</title>
<style>
  body { font-family: system-ui; max-width: 480px; margin: 2rem auto; padding: 0 1rem; }
  a.btn { display: inline-block; padding: .75rem 1.5rem; background: #0066cc;
          color: #fff; border-radius: 8px; text-decoration: none; margin: 1rem 0; }
  ol { line-height: 1.8; }
  code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; }
</style>
</head>
<body>
  <h1>noytads CA Certificate</h1>
  <p>Install this certificate to enable ad blocking in Safari on iOS.</p>
  <a class="btn" href="/ca.pem">Download CA Certificate</a>
  <h2>Installation steps</h2>
  <ol>
    <li>Tap the download link above</li>
    <li>Go to <b>Settings → General → VPN & Device Management</b></li>
    <li>Tap <b>noytads CA</b> → Install</li>
    <li>Go to <b>Settings → General → About → Certificate Trust Settings</b></li>
    <li>Enable full trust for <b>noytads CA</b></li>
  </ol>
  <p><small>The CA is unique to your router. Never share <code>ca.key</code>.</small></p>
</body>
</html>`)
	})

	log.Printf("CA export server on %s (open http://<router-ip>%s in Safari)", addr, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("CA export server error: %v", err)
	}
}
