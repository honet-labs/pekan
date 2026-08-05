package tests

import (
	"bytes"
	"context"
	"io"
	"testing"

	"pekan/backend/internal/platform/storage"
)

func TestLocalStoragePutOpenDelete(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	provider := storage.NewLocalStorage(basePath)

	putResult, err := provider.Put(context.Background(), storage.PutObjectInput{
		TenantID:    "tenant-1",
		Module:      "finance.transactions",
		ObjectKey:   "tenant-1/finance/receipt.txt",
		ContentType: "text/plain",
		Body:        bytes.NewBufferString("hello-storage"),
	})
	if err != nil {
		t.Fatalf("put object error: %v", err)
	}
	if putResult.Provider != "local" {
		t.Fatalf("unexpected provider: %s", putResult.Provider)
	}

	reader, err := provider.Open(context.Background(), storage.GetObjectInput{
		ObjectKey: putResult.ObjectKey,
	})
	if err != nil {
		t.Fatalf("open object error: %v", err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read object error: %v", err)
	}
	if string(raw) != "hello-storage" {
		t.Fatalf("unexpected content: %s", string(raw))
	}

	if err := provider.Delete(context.Background(), storage.GetObjectInput{
		ObjectKey: putResult.ObjectKey,
	}); err != nil {
		t.Fatalf("delete object error: %v", err)
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	provider := storage.NewLocalStorage(t.TempDir())
	_, err := provider.Put(context.Background(), storage.PutObjectInput{
		TenantID:    "tenant-1",
		Module:      "finance.transactions",
		ObjectKey:   "../escape.txt",
		ContentType: "text/plain",
		Body:        bytes.NewBufferString("unsafe"),
	})
	if err == nil {
		t.Fatalf("expected error for path traversal object key")
	}
}
