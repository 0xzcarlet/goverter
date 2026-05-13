package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type SavedFile struct {
	OriginalName string
	StorageKey   string
	AbsPath      string
	SizeBytes    int64
	ChecksumSHA  string
}

type Local struct {
	root       string
	uploadsDir string
	outputsDir string
	tmpDir     string
}

var ErrInvalidStorageKey = errors.New("invalid storage key")

func New(root, uploadsDir, outputsDir, tmpDir string) *Local {
	return &Local{
		root:       root,
		uploadsDir: uploadsDir,
		outputsDir: outputsDir,
		tmpDir:     tmpDir,
	}
}

func (l *Local) EnsureDirs() error {
	for _, dir := range []string{l.root, l.uploadsDir, l.outputsDir, l.tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (l *Local) SaveUpload(ctx context.Context, originalName string, source io.Reader) (SavedFile, error) {
	safeExt := strings.ToLower(filepath.Ext(originalName))
	fileID := uuid.NewString()
	storageKey := filepath.ToSlash(filepath.Join("uploads", fileID+safeExt))
	absPath := filepath.Join(l.root, filepath.FromSlash(storageKey))

	tmpPath := filepath.Join(l.tmpDir, fileID+".upload")
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return SavedFile{}, err
	}

	dst, err := os.Create(tmpPath)
	if err != nil {
		return SavedFile{}, err
	}
	defer dst.Close()

	hash := sha256.New()
	written, err := io.Copy(dst, io.TeeReader(source, hash))
	if err != nil {
		_ = os.Remove(tmpPath)
		return SavedFile{}, err
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(tmpPath)
		return SavedFile{}, err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return SavedFile{}, err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return SavedFile{}, err
	}

	return SavedFile{
		OriginalName: originalName,
		StorageKey:   storageKey,
		AbsPath:      absPath,
		SizeBytes:    written,
		ChecksumSHA:  hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (l *Local) PrepareOutputPath(targetFormat string) (SavedFile, error) {
	ext := "." + strings.TrimPrefix(strings.ToLower(targetFormat), ".")
	fileID := uuid.NewString()
	storageKey := filepath.ToSlash(filepath.Join("outputs", fileID+ext))
	absPath := filepath.Join(l.root, filepath.FromSlash(storageKey))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return SavedFile{}, err
	}
	return SavedFile{
		OriginalName: fmt.Sprintf("%s%s", fileID, ext),
		StorageKey:   storageKey,
		AbsPath:      absPath,
	}, nil
}

func (l *Local) AbsPath(storageKey string) string {
	return filepath.Join(l.root, filepath.FromSlash(storageKey))
}

func (l *Local) Open(storageKey string) (*os.File, error) {
	path, err := l.resolvePath(storageKey)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (l *Local) Remove(storageKey string) error {
	path, err := l.resolvePath(storageKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) resolvePath(storageKey string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(storageKey))
	if cleanKey == "." || cleanKey == "" {
		return "", ErrInvalidStorageKey
	}

	root := filepath.Clean(l.root)
	path := filepath.Join(root, cleanKey)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", ErrInvalidStorageKey
	}
	return path, nil
}
