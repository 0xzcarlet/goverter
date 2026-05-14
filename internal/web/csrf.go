package web

import (
	"net/url"
	"strings"

	"github.com/alexanderzull/file-converter/internal/config"
)

func csrfTrustedOrigins(cfg config.Config) []string {
	seen := map[string]struct{}{}
	add := func(origin string) {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return
		}
		seen[origin] = struct{}{}
	}

	add("localhost:8081")
	add("127.0.0.1:8081")

	if parsed, err := url.Parse(cfg.AppBaseURL); err == nil && parsed.Host != "" {
		add(parsed.Host)
	}

	out := make([]string, 0, len(seen))
	for origin := range seen {
		out = append(out, origin)
	}
	return out
}
