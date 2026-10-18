#!/bin/bash
# Smoke test: verify noytads injects scriptlets into YouTube HTML.
# Usage: ./scripts/test-injection.sh [proxy_host:port]
#
# Requires curl with proxy support. The CA must already be generated.

set -euo pipefail

PROXY="${1:-localhost:8443}"
CA="${2:-./ca.pem}"

echo "=== noytads injection smoke test ==="
echo "Proxy: $PROXY"
echo "CA:    $CA"
echo ""

# Test 1: Check proxy is reachable
echo -n "1. Proxy reachable... "
if curl -sk --connect-timeout 3 --proxy "http://$PROXY" https://www.google.com/ > /dev/null 2>&1; then
  echo "✅"
else
  echo "❌ Cannot connect to proxy at $PROXY"
  exit 1
fi

# Test 2: Fetch YouTube through proxy and check for injection
echo -n "2. Fetching youtube.com via proxy... "
RESPONSE=$(curl -sk --proxy "http://$PROXY" --cacert "$CA" \
  -H "User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)" \
  https://www.youtube.com/ 2>/dev/null)

if [ -z "$RESPONSE" ]; then
  echo "❌ Empty response"
  exit 1
fi
echo "✅ ($(echo "$RESPONSE" | wc -c | tr -d ' ') bytes)"

# Test 3: Check for scriptlet injection
echo -n "3. Scriptlet injection... "
if echo "$RESPONSE" | grep -q "noytads-scriptlets"; then
  echo "✅"
else
  echo "❌ <script id=\"noytads-scriptlets\"> not found"
  exit 1
fi

# Test 4: Check for cosmetic CSS injection
echo -n "4. Cosmetic CSS injection... "
if echo "$RESPONSE" | grep -q "noytads-cosmetic"; then
  echo "✅"
else
  echo "❌ <style id=\"noytads-cosmetic\"> not found"
  exit 1
fi

# Test 5: Check key scriptlet content
echo -n "5. json-prune present... "
if echo "$RESPONSE" | grep -q "adPlacements"; then
  echo "✅"
else
  echo "❌ json-prune code not found"
  exit 1
fi

# Test 6: Non-YouTube traffic should NOT be modified
echo -n "6. Non-target bypass... "
NONTARGET=$(curl -sk --proxy "http://$PROXY" --cacert "$CA" \
  https://example.com/ 2>/dev/null)
if echo "$NONTARGET" | grep -q "noytads-scriptlets"; then
  echo "❌ example.com was modified (should be bypassed)"
  exit 1
else
  echo "✅"
fi

echo ""
echo "=== All checks passed ==="
