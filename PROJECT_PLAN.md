# noytads — OpenWrt MITM Ad-Blocking Proxy

> Brave-style YouTube ad blocking (cosmetic filtering + scriptlet injection) as a transparent network proxy for OpenWrt.

---

## Goal

An OpenWrt package (`opkg`) that acts as a transparent HTTPS proxy and modifies HTML responses with CSS (cosmetic filtering) and JavaScript (scriptlet injection) — analogous to the three layers Brave browser uses. Primary use case: blocking YouTube ads in iOS Safari / PWA.

## Scope

### In Scope

- Transparent MITM proxy with dynamic certificate generation
- HTML response injection (CSS + JS) for arbitrary domains
- YouTube-specific scriptlets: `json-prune`, `fetch` proxy, `MutationObserver`
- uBlock/Brave-compatible filter list syntax (subset)
- Automatic filter list updates via cron
- UCI configuration + procd init script
- LuCI status page (optional, Phase 4)
- QUIC blocking via firewall rule

### Out of Scope

- Native iOS YouTube app (certificate pinning → not bypassable)
- Full uBlock Origin compatibility (only relevant subset)
- Video segment filtering (SponsorBlock-style)
- Custom DNS filtering (handled by AdGuard Home)

---

## Technology Stack

| Component         | Technology                 | Rationale                                              |
| ----------------- | -------------------------- | ------------------------------------------------------ |
| Proxy core        | Go 1.22+                   | Single binary, cross-compile for MIPS/ARM, low RAM     |
| MITM library      | `elazarl/goproxy`          | Battle-tested, MITM support, actively maintained       |
| TLS/certificates  | Go `crypto/x509`           | Dynamic cert generation without OpenSSL dependency     |
| Filter parser     | Custom (Go)                | Only subset needed, no external dependency             |
| Config            | UCI (OpenWrt standard)     | Integration into the OpenWrt ecosystem                 |
| Init              | procd                      | OpenWrt standard process manager                       |
| Build             | OpenWrt SDK / Buildroot    | Produces `.ipk` package                                |
| CI                | GitHub Actions             | Cross-compile + package build                          |

---

## Phases

### Phase 1 — Proof of Concept (Week 1–2)

**Goal:** Validate that the json-prune technique can eliminate YouTube ads in Safari via a network proxy.

**Deliverables:**

- [ ] Go binary listening as MITM proxy on `:8443`
- [ ] Self-signed root CA generated on first run
- [ ] Dynamic TLS certificates per domain (in-memory cache)
- [ ] Hardcoded YouTube-only: HTML responses from `*.youtube.com` are modified
- [ ] Injection of json-prune scriptlet + fetch proxy + MutationObserver
- [ ] Injection of base CSS for known YouTube ad containers
- [ ] Manual testing on iOS Safari with installed CA profile
- [ ] QUIC block rule documented

**Files:**

```
cmd/noytads/main.go             # Entry point, flag parsing, proxy start
internal/proxy/server.go        # goproxy setup, TLS config, listener
internal/proxy/ca.go            # CA generation, cert cache
internal/proxy/handler.go       # Response handler: content-type check, domain match
internal/proxy/inject.go        # HTML injection: <style> + <script> before </head>
internal/scriptlets/youtube.go  # Hardcoded YouTube scriptlets as Go string literals
```

**Acceptance Criteria:**

1. iOS Safari with CA profile → open youtube.com → no pre-roll/mid-roll ads
2. No TLS errors in Safari (certificate is accepted)
3. YouTube navigation (SPA) works after injection
4. Non-YouTube traffic is passed through unmodified
5. RAM usage < 30 MB during normal browsing

---

### Phase 2 — Rule Engine (Week 3–4)

**Goal:** Generic filter list support instead of hardcoded YouTube rules.

**Deliverables:**

- [ ] Parser for uBlock/Brave filter lists (relevant subset)
- [ ] Cosmetic filter rules (`domain##selector`) → CSS generation per domain
- [ ] Scriptlet rules (`domain##+js(name, args)`) → JS code resolution
- [ ] Scriptlet registry with implementations: `json-prune`, `set-constant`, `rpnt`, `addEventListener-defuser`, `no-setTimeout-if`
- [ ] Network block rules (`||domain^`) → connection reset or AdGuard export
- [ ] Domain matching with wildcard support (`youtube.*`, `*.googlevideo.com`)
- [ ] In-memory rule index with O(1) domain lookup (map-based)
- [ ] Load filter lists from local files

**Files:**

```
internal/rules/parser.go        # Filter list parser (line-by-line)
internal/rules/types.go         # CosmeticRule, ScriptletRule, NetworkRule structs
internal/rules/index.go         # DomainIndex: map[string][]Rule for fast lookup
internal/rules/matcher.go       # URL/domain matching logic
internal/scriptlets/registry.go # Scriptlet name → JS template mapping
internal/scriptlets/jsonprune.go
internal/scriptlets/setconstant.go
internal/scriptlets/rpnt.go
internal/scriptlets/templates/  # .js template files (embedded via go:embed)
```

**Acceptance Criteria:**

1. `brave-unbreak.txt` is correctly parsed (>95% of rules recognized)
2. YouTube ads are still blocked (regression test)
3. Cosmetic rules work on at least 3 other domains
4. Parse errors are logged but do not crash the proxy

---

### Phase 3 — OpenWrt Packaging (Week 5–6)

**Goal:** A clean `.ipk` package deployable via `opkg install`.

**Deliverables:**

- [ ] OpenWrt Makefile (SDK-compatible)
- [ ] UCI config schema (`/etc/config/noytads`)
- [ ] procd init script (`/etc/init.d/noytads`)
- [ ] Hotplug script for automatic iptables rules on interface up
- [ ] CA export endpoint: small HTTP server on `:8080/ca.pem` for easy iOS installation
- [ ] Log file rotation via `logrotate` or procd stdout
- [ ] Cross-compile for: `mips_24kc`, `aarch64_cortex-a53`, `x86_64`
- [ ] GitHub Actions workflow for automatic package build

**Files:**

```
openwrt/
├── Makefile                         # OpenWrt package Makefile
├── files/
│   ├── etc/
│   │   ├── config/noytads           # UCI defaults
│   │   ├── init.d/noytads           # procd init script
│   │   ├── hotplug.d/iface/         # iptables setup on interface up
│   │   └── noytads/
│   │       └── rules/               # Default filter lists (shipped)
│   └── usr/bin/noytads              # Placeholder (binary built by Makefile)
.github/
└── workflows/
    └── build.yml                    # Cross-compile + ipk build
```

**UCI Config:**

```uci
config noytads 'main'
    option enabled '1'
    option listen_port '8443'
    option ca_path '/etc/noytads/ca.pem'
    option ca_key_path '/etc/noytads/ca.key'
    option ca_export_port '8080'
    option log_level 'warn'
    option cert_cache_size '256'
    list bypass_domain 'banking.example.com'
    list bypass_domain '*.my-broker.com'

config filterlist
    option name 'brave-unbreak'
    option url 'https://raw.githubusercontent.com/nicil/brave-unbreak.txt'
    option enabled '1'
    option update_interval '6h'

config filterlist
    option name 'cosmetic-youtube'
    option url 'https://raw.githubusercontent.com/nicil/yt-cosmetic.txt'
    option enabled '1'
    option update_interval '12h'
```

**Acceptance Criteria:**

1. `opkg install noytads_*.ipk` works on OpenWrt 23.05+
2. `service noytads start/stop/restart` via procd
3. iptables rules are set automatically at boot
4. iOS can install CA via `http://<router-ip>:8080/ca.pem`
5. Binary size < 10 MB (stripped, UPX-compressed)

---

### Phase 4 — Filter List Updater + Hardening (Week 7–8)

**Deliverables:**

- [ ] Cron-based filter list updater with ETag/If-Modified-Since
- [ ] Atomic file swap on update (no temporary gap in rules)
- [ ] Hot-reload: proxy loads new rules without restart (SIGHUP or fsnotify)
- [ ] Bypass list: domains that are never intercepted (banking, IBKR, etc.)
- [ ] Rate limiting: max connections per client IP
- [ ] Graceful shutdown on SIGTERM
- [ ] Prometheus metrics endpoint (optional): blocked/modified/passthrough counters
- [ ] LuCI status page (optional): active rules, last updates, block counter

**Files:**

```
internal/updater/fetcher.go    # HTTP client with ETag caching
internal/updater/scheduler.go  # Cron-like scheduling logic
internal/proxy/bypass.go       # Domain bypass logic
internal/proxy/metrics.go      # Counters for blocked/modified/passed
```

---

## Project Structure (Full Picture)

```
noytads/
├── README.md
├── PROJECT_PLAN.md
├── LICENSE                          # MIT
├── go.mod
├── go.sum
│
├── cmd/
│   └── noytads/
│       └── main.go                  # Entry point
│
├── internal/
│   ├── config/
│   │   └── uci.go                   # UCI config parser (for OpenWrt)
│   │   └── config.go                # Config struct + defaults
│   │
│   ├── proxy/
│   │   ├── server.go                # goproxy setup, listener
│   │   ├── ca.go                    # CA generation, cert cache
│   │   ├── handler.go               # Request/response interception
│   │   ├── inject.go                # HTML modification pipeline
│   │   ├── bypass.go                # Domain bypass logic
│   │   └── metrics.go               # Counters
│   │
│   ├── rules/
│   │   ├── parser.go                # Filter list parser
│   │   ├── types.go                 # Rule type definitions
│   │   ├── index.go                 # Domain → rules index
│   │   └── matcher.go               # URL/domain matching
│   │
│   ├── scriptlets/
│   │   ├── registry.go              # Name → implementation mapping
│   │   ├── jsonprune.go             # json-prune scriptlet
│   │   ├── setconstant.go           # set-constant scriptlet
│   │   ├── rpnt.go                  # replace-node-text scriptlet
│   │   └── templates/               # Embedded JS templates
│   │       ├── json_prune.js
│   │       ├── fetch_proxy.js
│   │       ├── mutation_observer.js
│   │       └── set_constant.js
│   │
│   └── updater/
│       ├── fetcher.go               # HTTP fetch with ETag
│       └── scheduler.go             # Periodic update logic
│
├── openwrt/
│   ├── Makefile                     # OpenWrt SDK package Makefile
│   └── files/
│       └── etc/
│           ├── config/noytads
│           ├── init.d/noytads
│           └── hotplug.d/iface/50-noytads
│
├── scripts/
│   ├── install-ca-ios.md            # iOS CA installation guide
│   └── test-injection.sh            # Curl-based smoke test
│
├── testdata/
│   ├── brave-unbreak-sample.txt     # Subset for testing
│   └── youtube-response.html        # Sample YouTube HTML for unit tests
│
└── .github/
    └── workflows/
        └── build.yml                # CI: test + cross-compile + ipk
```

---

## Key Decisions

### Why Go and not Rust?

- Simpler cross-compilation for OpenWrt targets (`GOOS=linux GOARCH=mipsle`)
- `goproxy` is battle-tested for MITM; Rust equivalents (`hudsucker`) are less mature
- Go is more accessible given a Python background
- Trade-off: slightly higher RAM than Rust, but still < 30 MB

### Why `goproxy` and not `mitmproxy`?

- `mitmproxy` is Python → too heavy for OpenWrt (RAM, disk, runtime)
- `goproxy` compiles into a single static binary
- Sufficient for the use case (response body modification)

### Filter List Compatibility (intentionally limited)

Only this subset of uBlock syntax is supported:

| Syntax                          | Type             | Supported   |
| ------------------------------- | ---------------- | ----------- |
| `\|\|domain^`                   | Network block    | ✅           |
| `domain##selector`              | Cosmetic hide    | ✅           |
| `domain##+js(scriptlet, args)`  | Scriptlet inject | ✅           |
| `domain#@#selector`             | Cosmetic unhide  | ✅           |
| `@@\|\|domain^`                 | Network allow    | ✅           |
| `domain$removeparam=...`        | Param removal    | ❌ Phase 5   |
| `domain$redirect=...`           | Redirect         | ❌ Phase 5   |
| Regex-based rules               | All              | ❌           |

### Response Injection Strategy

Injection happens via **streaming rewrite** of the response body:

1. Check `Content-Type: text/html`
2. Read response body (with size limit: 5 MB)
3. Find `</head>` position (byte scan, no HTML parser needed)
4. Write: everything before `</head>` → injected CSS + JS → `</head>` → rest
5. Update `Content-Length` header (or remove it for chunked encoding)

No full HTML parser needed — `</head>` is a reliable injection point.

### Certificate Caching

- LRU cache for generated certificates (default: 256 entries)
- Avoids repeated RSA key generation (expensive on MIPS)
- Alternative: ECDSA P-256 keys (faster to generate)

---

## Dependencies (Go Modules)

```
require (
    github.com/elazarl/goproxy v0.0.0-latest   // MITM proxy
)
```

Intentionally minimal dependencies. Everything else (config, filter parsing, scriptlets) is custom-built to keep the binary small.

---

## Risks

| Risk                                     | Impact | Mitigation                                              |
| ---------------------------------------- | ------ | ------------------------------------------------------- |
| YouTube changes player API structure     | High   | Update filter lists regularly, monitor Brave repo       |
| Safari starts cert pinning for YT        | High   | No workaround possible — end of project for iOS Safari  |
| goproxy has bugs with HTTP/2 responses   | Medium | Fallback to HTTP/1.1 via ALPN negotiation               |
| OpenWrt router hardware too weak for TLS | Medium | ECDSA instead of RSA, cert cache, connection pooling    |
| QUIC fallback bypasses proxy             | Medium | iptables REJECT for UDP 443                             |

---

## Quick Start (Phase 1)

```bash
# Initialize repository
mkdir noytads && cd noytads
git init
go mod init github.com/caf3babe/noytads

# Fetch dependency
go get github.com/elazarl/goproxy

# Entry point: cmd/noytads/main.go
# → Start proxy, generate CA, modify YouTube responses

# Run
go run ./cmd/noytads -listen :8443 -ca-path ./ca.pem
# On iOS: install CA, configure proxy, open youtube.com
```
