package proxy

import (
	"bytes"
	"fmt"

	"github.com/caf3babe/noytads/internal/scriptlets"
)

// InjectIntoHTML inserts cosmetic CSS and scriptlet JS before </head>.
// Returns nil if no injection is needed or </head> is not found.
func InjectIntoHTML(html []byte, host string) []byte {
	// Find </head> injection point (case-insensitive)
	headClose := findHeadClose(html)
	if headClose == -1 {
		return nil
	}

	css := scriptlets.GetCosmeticCSS(host)
	js := scriptlets.GetScriptlets(host)

	if css == "" && js == "" {
		return nil
	}

	var injection bytes.Buffer

	if css != "" {
		injection.WriteString(fmt.Sprintf(
			"\n<style id=\"noytads-cosmetic\">\n%s\n</style>\n", css))
	}
	if js != "" {
		injection.WriteString(fmt.Sprintf(
			"\n<script id=\"noytads-scriptlets\">\n%s\n</script>\n", js))
	}

	// Build modified HTML: [before </head>] + [injection] + [</head> onwards]
	result := make([]byte, 0, len(html)+injection.Len())
	result = append(result, html[:headClose]...)
	result = append(result, injection.Bytes()...)
	result = append(result, html[headClose:]...)

	return result
}

// findHeadClose returns the byte offset of "</head>" (case-insensitive).
func findHeadClose(html []byte) int {
	// Try exact match first (most common)
	if idx := bytes.Index(html, []byte("</head>")); idx != -1 {
		return idx
	}
	// Fallback: case-insensitive search
	lower := bytes.ToLower(html)
	return bytes.Index(lower, []byte("</head>"))
}
