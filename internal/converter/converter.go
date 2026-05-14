package converter

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Service struct {
	execName string
}

const (
	FormatPDF  = "pdf"
	FormatEPUB = "epub"

	MIMEPDF                    = "application/pdf"
	MIMEEPUB                   = "application/epub+zip"
	MIMEApplicationOctetStream = "application/octet-stream"
)

func New(execName string) *Service {
	if execName == "" {
		execName = "ebook-convert"
	}
	return &Service{execName: execName}
}

func (s *Service) Check() error {
	_, err := exec.LookPath(s.execName)
	return err
}

func (s *Service) Version(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, s.execName, "--version")
	return cmd.CombinedOutput()
}

func (s *Service) Convert(ctx context.Context, inputPath, outputPath string) error {
	target := strings.TrimPrefix(strings.ToLower(filepath.Ext(outputPath)), ".")
	if err := ValidatePair(inputPath, target); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, s.execName, inputPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ebook-convert failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func ValidatePair(inputPath, targetFormat string) error {
	source := strings.TrimPrefix(strings.ToLower(filepath.Ext(inputPath)), ".")
	target := strings.TrimPrefix(strings.ToLower(targetFormat), ".")

	switch {
	case source == FormatPDF && target == FormatEPUB:
		return nil
	case source == FormatEPUB && target == FormatPDF:
		return nil
	case source == "":
		return errors.New("source format is missing")
	case target == "":
		return errors.New("target format is missing")
	default:
		return fmt.Errorf("unsupported conversion: %s -> %s", source, target)
	}
}

func MIMEByFormat(format string) string {
	switch strings.TrimPrefix(strings.ToLower(format), ".") {
	case FormatPDF:
		return MIMEPDF
	case FormatEPUB:
		return MIMEEPUB
	default:
		return MIMEApplicationOctetStream
	}
}
