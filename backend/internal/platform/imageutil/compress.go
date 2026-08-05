package imageutil

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	// Register image decoders
	_ "image/jpeg"
	_ "image/png"
)

// CompressImage reads the image from r, resizes it if it exceeds maxDim (e.g., 2000px)
// while keeping the aspect ratio, and encodes it back to its original format.
// If compression fails or the format is not JPEG/PNG, it returns the original bytes.
func CompressImage(r io.Reader, mimeType string) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType != "image/jpeg" && mimeType != "image/jpg" && mimeType != "image/png" {
		return data, nil
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Fall back to original data if decoding fails
		return data, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Use a high limit like 2000px to ensure we don't degrade quality for large photos
	maxDim := 2000
	var newImg image.Image = img

	if width > maxDim || height > maxDim {
		var newW, newH int
		if width > height {
			newW = maxDim
			newH = (height * maxDim) / width
		} else {
			newH = maxDim
			newW = (width * maxDim) / height
		}

		// Resize image using nearest-neighbor / bilinear-like mapping
		rgba := image.NewRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				rgba.Set(x, y, img.At(x*width/newW, y*height/newH))
			}
		}
		newImg = rgba
	}

	var buf bytes.Buffer
	if format == "png" {
		// Lossless PNG compression
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, newImg); err != nil {
			return data, nil
		}
	} else {
		// JPEG compression (Quality 80 is visually indistinguishable but yields high compression)
		if err := jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 80}); err != nil {
			return data, nil
		}
	}

	// Only return the compressed version if it actually reduced the size
	if buf.Len() < len(data) {
		return buf.Bytes(), nil
	}
	return data, nil
}
