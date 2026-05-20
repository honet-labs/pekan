package filescan

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
)

const (
	maxScanReadBytes = 12 * 1024 * 1024
	eicarSignature   = "EICAR-STANDARD-ANTIVIRUS-TEST-FILE"
)

var allowedScanMimeTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
	"application/pdf": {},
}

type SignatureScanner struct{}

func NewSignatureScanner() *SignatureScanner {
	return &SignatureScanner{}
}

func (s *SignatureScanner) Scan(_ context.Context, in ScanInput) (string, error) {
	mime := strings.ToLower(strings.TrimSpace(strings.Split(in.MimeType, ";")[0]))
	if _, ok := allowedScanMimeTypes[mime]; !ok {
		return ScanStatusInfected, nil
	}

	limitedReader := io.LimitReader(in.Reader, maxScanReadBytes+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return ScanStatusFailed, err
	}
	if len(body) == 0 {
		return ScanStatusInfected, nil
	}
	if len(body) > maxScanReadBytes {
		return ScanStatusFailed, errors.New("scan body exceeds max scanner bytes")
	}

	normalized := bytes.ToUpper(body)
	if bytes.Contains(normalized, []byte(eicarSignature)) {
		return ScanStatusInfected, nil
	}

	return ScanStatusClean, nil
}
