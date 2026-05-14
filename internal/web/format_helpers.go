package web

import (
	"path/filepath"
	"strings"

	"github.com/alexanderzull/file-converter/internal/converter"
)

func normalizeFormat(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func detectMimeType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case "." + converter.FormatPDF:
		return converter.MIMEPDF
	case "." + converter.FormatEPUB:
		return converter.MIMEEPUB
	default:
		return converter.MIMEApplicationOctetStream
	}
}

func buildDownloadFilename(sourceName, targetFormat string) string {
	base := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	base = sanitizeFilename(base)
	if base == "" {
		base = "converted-file"
	}
	target := strings.TrimPrefix(strings.ToLower(targetFormat), ".")
	if target == "" {
		target = "bin"
	}
	return base + "." + target
}

func sanitizeFilename(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "\x00", "", "\n", " ", "\r", " ", "\"", "", "'", "")
	safe := strings.TrimSpace(replacer.Replace(value))
	safe = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case strings.ContainsRune(" ._-()", r):
			return r
		default:
			return '-'
		}
	}, safe)
	safe = strings.Trim(safe, ". -_")
	return safe
}
