package scriptlets

import "strings"

// GetCosmeticCSS returns cosmetic filtering CSS rules for the given host.
// Phase 1: hardcoded YouTube selectors.
// Phase 2: driven by parsed filter lists via the rule engine.
func GetCosmeticCSS(host string) string {
	if !isYouTube(host) {
		return ""
	}

	return `.ytp-ad-overlay-container,
.ytp-ad-text-overlay,
.ytp-ad-skip-button-container,
.ytp-ad-message-container,
.ytp-ad-persistent-progress-bar-container,
.ytp-ad-image-overlay,
#player-ads,
#masthead-ad,
ytd-promoted-sparkles-web-renderer,
ytd-display-ad-renderer,
ytd-ad-slot-renderer,
ytd-in-feed-ad-layout-renderer,
ytd-banner-promo-renderer,
.ytd-mealbar-promo-renderer,
tp-yt-paper-dialog:has(#sponsor-link) {
  display: none !important;
}`
}

// GetScriptlets returns JavaScript to inject for the given host.
// This implements Brave's Layer 3: json-prune, fetch proxy, and MutationObserver.
func GetScriptlets(host string) string {
	if !isYouTube(host) {
		return ""
	}

	return `(function() {
  'use strict';

  // ================================================================
  // json-prune: Strip ad data from all parsed JSON objects.
  // YouTube's player loads ad config from playerResponse JSON.
  // By pruning these keys, the player never sees ad instructions.
  // ================================================================
  const AD_KEYS = ['adPlacements', 'playerAds', 'adSlots'];

  const _origParse = JSON.parse;
  JSON.parse = function(text, reviver) {
    const obj = _origParse.call(this, text, reviver);
    if (obj && typeof obj === 'object') {
      pruneAds(obj);
    }
    return obj;
  };

  function pruneAds(obj) {
    for (const key of AD_KEYS) {
      if (key in obj) {
        obj[key] = [];
      }
      // YouTube nests ad data under playerResponse in some code paths
      if (obj.playerResponse && key in obj.playerResponse) {
        obj.playerResponse[key] = [];
      }
    }
  }

  // ================================================================
  // fetch proxy: Intercept /player?key= API responses.
  // This catches the primary YouTube player config endpoint.
  // ================================================================
  const _origFetch = window.fetch;
  window.fetch = function(...args) {
    let url = '';
    if (args[0] instanceof Request) {
      url = args[0].url;
    } else if (typeof args[0] === 'string') {
      url = args[0];
    } else if (args[0] instanceof URL) {
      url = args[0].href;
    }

    if (!url.includes('/player') || !url.includes('key=')) {
      return _origFetch.apply(this, args);
    }

    return _origFetch.apply(this, args).then(function(response) {
      return response.text().then(function(text) {
        try {
          const json = _origParse(text);
          pruneAds(json);
          return new Response(JSON.stringify(json), {
            status: response.status,
            statusText: response.statusText,
            headers: response.headers,
          });
        } catch (e) {
          // Not JSON — return unmodified
          return new Response(text, {
            status: response.status,
            statusText: response.statusText,
            headers: response.headers,
          });
        }
      });
    });
  };

  // ================================================================
  // XMLHttpRequest intercept: Fallback for legacy YT code paths
  // that still use XHR instead of fetch.
  // ================================================================
  const _origXHROpen = XMLHttpRequest.prototype.open;
  const _origXHRSend = XMLHttpRequest.prototype.send;

  XMLHttpRequest.prototype.open = function(method, url, ...rest) {
    this._noytads_url = typeof url === 'string' ? url : url?.toString?.() || '';
    return _origXHROpen.call(this, method, url, ...rest);
  };

  XMLHttpRequest.prototype.send = function(body) {
    if (this._noytads_url &&
        this._noytads_url.includes('/player') &&
        this._noytads_url.includes('key=')) {
      this.addEventListener('readystatechange', function() {
        if (this.readyState === 4) {
          try {
            const json = _origParse(this.responseText);
            pruneAds(json);
            Object.defineProperty(this, 'responseText', {
              value: JSON.stringify(json),
              writable: false,
              configurable: true,
            });
          } catch (e) {
            // Not JSON, skip
          }
        }
      });
    }
    return _origXHRSend.call(this, body);
  };

  // ================================================================
  // MutationObserver: Catch dynamically inserted ad containers.
  // YouTube is an SPA — new ad elements appear during navigation.
  // ================================================================
  const AD_SELECTORS = [
    '.ytp-ad-overlay-container',
    '.ytp-ad-text-overlay',
    'ytd-promoted-sparkles-web-renderer',
    'ytd-display-ad-renderer',
    'ytd-ad-slot-renderer',
    'ytd-in-feed-ad-layout-renderer',
    '.ytp-ad-skip-button-container',
    '.ytp-ad-message-container',
    '.ytp-ad-persistent-progress-bar-container',
  ].join(',');

  function hideAdNodes(root) {
    if (!root || !root.querySelectorAll) return;
    root.querySelectorAll(AD_SELECTORS).forEach(function(el) {
      el.style.setProperty('display', 'none', 'important');
    });
  }

  // Observe the entire document for new ad nodes
  const observer = new MutationObserver(function(mutations) {
    for (const mut of mutations) {
      for (const node of mut.addedNodes) {
        if (node.nodeType !== 1) continue;
        if (node.matches && node.matches(AD_SELECTORS)) {
          node.style.setProperty('display', 'none', 'important');
        }
        hideAdNodes(node);
      }
    }
  });

  // Start observing as early as possible
  if (document.documentElement) {
    observer.observe(document.documentElement, { childList: true, subtree: true });
  }

  // Also clean up anything already in the DOM
  if (document.body) {
    hideAdNodes(document.body);
  }

  console.log('[noytads] YouTube ad-blocking scriptlets active');
})();`
}

func isYouTube(host string) bool {
	host = strings.ToLower(host)
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.HasSuffix(host, "youtube.com") ||
		strings.HasSuffix(host, "youtubekids.com") ||
		strings.HasSuffix(host, "youtube-nocookie.com")
}
