package flow

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResizeImageForGateway_SmallImageUnchanged(t *testing.T) {
	t.Parallel()
	// 100×100 PNG — well under limits.
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	data := buf.Bytes()

	out, mime := resizeImageForGateway(data, "image/png")
	assert.Equal(t, data, out, "small image should be returned as-is")
	assert.Equal(t, "image/png", mime)
}

func TestResizeImageForGateway_LargeImageResized(t *testing.T) {
	t.Parallel()
	// 3000×2000 JPEG — exceeds 1568 dimension limit.
	img := image.NewRGBA(image.Rect(0, 0, 3000, 2000))
	// Fill with non-uniform data so JPEG doesn't compress to nothing.
	for y := 0; y < 2000; y++ {
		for x := 0; x < 3000; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}))
	data := buf.Bytes()
	require.Greater(t, len(data), 100000, "test image should be non-trivial")

	out, mime := resizeImageForGateway(data, "image/jpeg")
	assert.Less(t, len(out), len(data), "resized image should be smaller")
	assert.Equal(t, "image/jpeg", mime)

	// Verify dimensions.
	decoded, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.LessOrEqual(t, decoded.Width, gatewayImageMaxDimension)
	assert.LessOrEqual(t, decoded.Height, gatewayImageMaxDimension)
}

func TestResizeImageForGateway_NonImagePassthrough(t *testing.T) {
	t.Parallel()
	data := []byte("not an image")
	out, mime := resizeImageForGateway(data, "application/pdf")
	assert.Equal(t, data, out)
	assert.Equal(t, "application/pdf", mime)
}

func TestResizeImageForGateway_PNGWithTransparencyKeepsPNG(t *testing.T) {
	t.Parallel()
	// 2000×2000 PNG with alpha — should stay PNG after resize.
	img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
	for y := 0; y < 2000; y++ {
		for x := 0; x < 2000; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: uint8(x % 256)})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	data := buf.Bytes()

	out, mime := resizeImageForGateway(data, "image/png")
	assert.Less(t, len(out), len(data))
	assert.Equal(t, "image/png", mime)
}
