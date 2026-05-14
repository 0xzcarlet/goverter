package db

import "time"

type CreateStoredFileParams struct {
	UserID       string
	OriginalName string
	StorageKey   string
	MimeType     string
	SizeBytes    int64
	ChecksumSHA  string
}

type StoredFile struct {
	ID           string
	UserID       string
	OriginalName string
	StorageKey   string
	MimeType     string
	SizeBytes    int64
	ChecksumSHA  *string
	CreatedAt    time.Time
}

type DailyUsage struct {
	UserID         string
	QuotaDate      time.Time
	ReservedCount  int
	CompletedCount int
}

type ConversionJob struct {
	ID               string
	UserID           string
	SourceFileID     string
	TargetFormat     string
	Status           string
	OutputFileID     *string
	OutputStorageKey *string
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	SourceFileName   string
}

type CreateQueuedConversionParams struct {
	UserID       string
	TargetFormat string
	QuotaDate    time.Time
	Limit        int
	SourceFile   CreateStoredFileParams
}

type ClaimedJob struct {
	ConversionJob
	QuotaDate        time.Time
	QuotaStatus      string
	SourceStorageKey string
	SourceMIMEType   string
}

type DownloadableFile struct {
	StorageKey   string
	MimeType     string
	TargetFormat string
	SourceName   string
	OutputName   string
	OutputFileID string
}
