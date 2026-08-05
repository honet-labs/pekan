package tests

import (
	"bytes"
	"context"
	"testing"

	"pekan/backend/internal/platform/filescan"
)

func TestSignatureScannerDetectsEICAR(t *testing.T) {
	t.Parallel()

	scanner := filescan.NewSignatureScanner()
	outcome, err := scanner.Scan(context.Background(), filescan.ScanInput{
		MimeType: "application/pdf",
		Reader: bytes.NewBufferString(
			"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*",
		),
	})
	if err != nil {
		t.Fatalf("unexpected scan error: %v", err)
	}
	if outcome != filescan.ScanStatusInfected {
		t.Fatalf("expected infected outcome, got=%s", outcome)
	}
}

func TestSignatureScannerMarksCleanForSafePayload(t *testing.T) {
	t.Parallel()

	scanner := filescan.NewSignatureScanner()
	outcome, err := scanner.Scan(context.Background(), filescan.ScanInput{
		MimeType: "application/pdf",
		Reader:   bytes.NewBufferString("%PDF-1.7 sample document"),
	})
	if err != nil {
		t.Fatalf("unexpected scan error: %v", err)
	}
	if outcome != filescan.ScanStatusClean {
		t.Fatalf("expected clean outcome, got=%s", outcome)
	}
}
