# noytads

Brave-style YouTube ad blocking as a transparent MITM proxy for OpenWrt.

Injects cosmetic CSS and JavaScript scriptlets (`json-prune`, `fetch` proxy, `MutationObserver`) into HTML responses — the same techniques Brave browser uses to strip YouTube ads at the player API level.

## How it works

| Layer | Technique | What it does |
|-------|-----------|--------------|
| 1 | DNS blocking (AdGuard Home) | Blocks known ad/tracker domains |
| 2 | Cosmetic CSS injection | Hides ad containers via `display: none` |
| 3 | Scriptlet JS injection | Strips ad data from YouTube's player API JSON |

The proxy intercepts HTTPS traffic via `iptables REDIRECT`, terminates TLS with a custom CA, inspects HTML responses, and injects blocking code before `</head>`.

## Status

🚧 **Phase 1: Proof of Concept** — hardcoded YouTube rules, no rule engine yet.

## Requirements

- Go 1.22+
- iOS device with custom CA profile installed
- OpenWrt router (for production) or any Linux box (for development)
- AdGuard Home for DNS-level blocking (Layer 1)

## Quick start (development)

```bash
# Build and run
go run ./cmd/noytads \
  -listen :8443 \
  -ca-path ./ca.pem \
  -ca-key ./ca.key \
  -log-level debug

# On first run, a CA is generated automatically.
```

### Install CA on iOS

1. Open `http://<your-machine-ip>:8080` in Safari on iOS
2. Download and install the CA certificate
3. Go to **Settings → General → About → Certificate Trust Settings**
4. Enable full trust for **noytads CA**

### Configure proxy

**Option A — Transparent proxy (production, requires iptables):**

```bash
# Redirect all HTTPS from LAN through the proxy
iptables -t nat -A PREROUTING -i br-lan -p tcp --dport 443 -j REDIRECT --to-port 8443
# Block QUIC so browsers fall back to TCP
iptables -A FORWARD -p udp --dport 443 -j REJECT
```

**Option B — Manual proxy (development):**
On iOS: Settings → Wi-Fi → your network → Configure Proxy → Manual → set to your machine IP, port 8443.

### Test it

Open youtube.com in Safari. Pre-roll and mid-roll ads should be gone.

```bash
# Smoke test via curl
./scripts/test-injection.sh 8443
```

## Project structure

```txt
cmd/noytads/              Entry point
internal/
  proxy/
    server.go             goproxy setup, CA export web server
    ca.go                 ECDSA CA generation, cert loading
    handler.go            Response interception, gzip handling
    inject.go             HTML injection (<style> + <script> before </head>)
  scriptlets/
    youtube.go            Hardcoded YouTube CSS + JS (Phase 1)
  rules/                  Filter list parser (Phase 2)
  updater/                Periodic list updates (Phase 4)
openwrt/                  OpenWrt packaging (init, UCI, iptables)
```

## Limitations

| Works | Doesn't work |
|-------|-------------|
| YouTube in Safari on iOS | Native YouTube iOS app (cert pinning) |
| YouTube as PWA (Add to Home Screen) | Apps with custom cert stores |
| All websites in Safari/Chrome iOS | HTTP/3 QUIC (must be blocked) |
| Desktop browsers on LAN | Sites enforcing HPKP |

## License

MIT
