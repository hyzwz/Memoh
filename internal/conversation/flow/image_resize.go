package flow

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
)

const (
	// Anthropic recommends images no larger than 1568×1568.
	// Resizing to this cap reduces tokens billed and keeps payloads small.
	gatewayImageMaxDimension = 1568
	// After resize, raw image bytes should stay under this limit to keep
	// the base64 payload (×4/3) reasonable for the model API and proxies.
	gatewayImageMaxRawBytes int64 = 1 * 1024 * 1024 // 1 MB
)

// resizeImageForGateway decodes an image, resizes it if it exceeds the
// dimension or byte-size thresholds, and re-encodes to JPEG.
// If the image is already small enough it is returned as-is.
// Non-image data or decode failures are returned unchanged.
func resizeImageForGateway(data []byte, mime string) ([]byte, string) {
	if len(data) == 0 {
		return data, mime
	}
	lower := strings.ToLower(strings.TrimSpace(mime))
	if !strings.HasPrefix(lower, "image/") {
		return data, mime
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Can't decode → return original.
		return data, mime
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	needsResize := w > gatewayImageMaxDimension || h > gatewayImageMaxDimension
	needsShrink := int64(len(data)) > gatewayImageMaxRawBytes

	if !needsResize && !needsShrink {
		return data, mime
	}

	// Calculate new dimensions preserving aspect ratio.
	newW, newH := w, h
	if needsResize {
		if w >= h {
			newW = gatewayImageMaxDimension
			newH = h * gatewayImageMaxDimension / w
		} else {
			newH = gatewayImageMaxDimension
			newW = w * gatewayImageMaxDimension / h
		}
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}
	}

	// Draw resized image using standard library bilinear-ish scaler.
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	if newW == w && newH == h {
		// Same dimensions but file too large — just re-encode at lower quality.
		draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)
	} else {
		// Scale: use draw.ApproxBiLinear if available; fall back to basic draw.
		scaleDown(dst, img)
	}

	// Re-encode. Prefer JPEG for photos (smaller), keep PNG for transparency.
	var buf bytes.Buffer
	outMime := mime

	switch {
	case format == "png" && hasTransparency(dst):
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, dst); err != nil {
			return data, mime
		}
		outMime = "image/png"
	default:
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
			return data, mime
		}
		outMime = "image/jpeg"
	}

	// If re-encoding somehow made it bigger, return original.
	if int64(buf.Len()) >= int64(len(data)) && !needsResize {
		return data, mime
	}

	return buf.Bytes(), outMime
}

// scaleDown performs nearest-neighbor downscaling using the standard library.
func scaleDown(dst *image.RGBA, src image.Image) {
	dstBounds := dst.Bounds()
	srcBounds := src.Bounds()
	dw := dstBounds.Dx()
	dh := dstBounds.Dy()
	sw := srcBounds.Dx()
	sh := srcBounds.Dy()

	for y := 0; y < dh; y++ {
		sy := srcBounds.Min.Y + y*sh/dh
		for x := 0; x < dw; x++ {
			sx := srcBounds.Min.X + x*sw/dw
			dst.Set(dstBounds.Min.X+x, dstBounds.Min.Y+y, src.At(sx, sy))
		}
	}
}

// hasTransparency checks if any pixel in the RGBA image has alpha < 255.
func hasTransparency(img *image.RGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] < 255 {
			return true
		}
	}
	return false
}

// readAndResizeImage reads image data from a reader, applies resizing if
// needed, and returns a new reader plus the (potentially updated) mime type.
func readAndResizeImage(reader io.Reader, maxBytes int64, attachmentType, mime string) (io.Reader, string, error) {
	if reader == nil {
		return nil, mime, fmt.Errorf("reader is required")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, mime, fmt.Errorf("read asset for resize: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, mime, fmt.Errorf("asset too large: %d > %d", len(data), maxBytes)
	}

	if strings.EqualFold(strings.TrimSpace(attachmentType), "image") {
		data, mime = resizeImageForGateway(data, mime)
	}

	return bytes.NewReader(data), mime, nil
}

// Ensure gif decoder is registered.
func init() {
	// image/png and image/jpeg are registered by their import.
	// gif needs an explicit import.
	_ = gif.Decode
}
