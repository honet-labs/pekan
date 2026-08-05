package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestCompressImage_JPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode dummy jpeg: %v", err)
	}

	compressed, err := CompressImage(&buf, "image/jpeg")
	if err != nil {
		t.Fatalf("failed to compress jpeg: %v", err)
	}

	if len(compressed) == 0 {
		t.Errorf("expected non-empty compressed bytes")
	}
}

func TestCompressImage_PNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode dummy png: %v", err)
	}

	compressed, err := CompressImage(&buf, "image/png")
	if err != nil {
		t.Fatalf("failed to compress png: %v", err)
	}

	if len(compressed) == 0 {
		t.Errorf("expected non-empty compressed bytes")
	}
}

func TestCompressImage_Unsupported(t *testing.T) {
	data := []byte("plain text data")
	compressed, err := CompressImage(bytes.NewReader(data), "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(compressed, data) {
		t.Errorf("expected original data to be returned for unsupported type")
	}
}
