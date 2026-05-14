package ui

import (
	"html/template"
	"path/filepath"
	"strings"
	"time"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(v *time.Time) string {
			if v == nil {
				return "—"
			}
			return v.Local().Format("02 Jan 2006 15:04")
		},
		"formatCreated": func(v time.Time) string {
			return v.Local().Format("02 Jan 2006 15:04")
		},
		"statusClass": func(status string) string {
			switch strings.ToLower(status) {
			case JobStatusDone:
				return "status status-done"
			case JobStatusFailed:
				return "status status-failed"
			case JobStatusProcessing:
				return "status status-processing"
			default:
				return "status status-queued"
			}
		},
		"statusText": func(status string) string {
			if status == "" {
				return "Unknown"
			}
			return strings.ToUpper(status[:1]) + strings.ToLower(status[1:])
		},
		"flashClass": func(kind string) string {
			switch strings.ToLower(kind) {
			case "error":
				return "flash flash-error"
			case "success":
				return "flash flash-success"
			default:
				return "flash"
			}
		},
		"downloadPath": func(jobID string) string {
			return "/app/conversions/" + jobID + "/download"
		},
		"fileStem": func(name string) string {
			base := filepath.Base(name)
			return strings.TrimSuffix(base, filepath.Ext(base))
		},
	}
}
