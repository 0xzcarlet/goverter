package ui

import (
	"html/template"
	"time"

	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/quota"
)

type Flash struct {
	Kind    string
	Message string
}

type Job struct {
	ID               string
	Status           string
	TargetFormat     string
	SourceFileName   string
	OutputStorageKey *string
	ErrorMessage     *string
	CreatedAt        time.Time
	FinishedAt       *time.Time
}

const (
	JobStatusDone       = "done"
	JobStatusFailed     = "failed"
	JobStatusProcessing = "processing"
)

type layoutData struct {
	Title       string
	AppName     string
	CurrentUser *auth.User
	CSRFToken   string
	Body        template.HTML
}

type authView struct {
	Flash     *Flash
	CSRFField template.HTML
	Email     string
}

type resetView struct {
	Flash      *Flash
	CSRFField  template.HTML
	HasSession bool
}

type dashboardView struct {
	Flash       *Flash
	CSRFField   template.HTML
	CurrentUser *auth.User
	Jobs        []Job
	MaxUploadMB int64
	Quota       quota.Summary
}

type callbackView struct {
	Heading    string
	Message    string
	RedirectTo string
	PostPath   string
	CSRFToken  string
}
