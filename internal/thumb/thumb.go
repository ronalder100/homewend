// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package thumb makes the small pictures the window's grid shows, and keeps
// them. Everything is Go but video: no cgo, so the app still cross-compiles.
//
// JPEG, PNG and GIF are the standard library's, WebP is x/image's, HEIC is
// gen2brain/heic (a Rust decoder run as WASM, or the system's libheif when
// there is one). A video's first frame is ffmpeg's, as every photo app we
// looked at does it: Ente, Immich, PhotoPrism, Photoview.
package thumb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/heic"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/ronalder100/homewend/internal/library"
)

// Size is the long edge of a thumbnail: the grid's tiles are about 104 to 180
// pixels, twice that on a high-density screen.
const Size = 256

// quality of the JPEG written; the size where artefacts stop showing at Size.
const quality = 80

// ErrNoDecoder is a file no decoder here reads: a video without ffmpeg, a
// format nobody knows.
var ErrNoDecoder = errors.New("no decoder for this file")

// Path is where the thumbnail of the photo with this hash is kept: by
// content, so it never goes stale and a moved photo keeps it.
func Path(cacheDir, hash string) string {
	if len(hash) < 2 {
		return filepath.Join(cacheDir, hash+".jpg")
	}
	return filepath.Join(cacheDir, hash[:2], hash+".jpg")
}

// Make writes the thumbnail of src at dst, whole or not at all.
func Make(ctx context.Context, src, dst string) error {
	img, err := decode(ctx, src)
	if err != nil {
		return err
	}
	img = upright(fit(img), library.Orientation(src))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

var videos = map[string]bool{".mp4": true, ".mov": true, ".m4v": true, ".3gp": true, ".avi": true, ".mkv": true, ".webm": true}

func decode(ctx context.Context, src string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(src))
	if videos[ext] {
		return frame(ctx, src)
	}
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if ext == ".heic" || ext == ".heif" {
		return heic.Decode(f)
	}
	img, _, err := image.Decode(f)
	if errors.Is(err, image.ErrFormat) {
		return nil, fmt.Errorf("%w: %s", ErrNoDecoder, ext)
	}
	return img, err
}

// frame is a video's first frame, already at Size, from ffmpeg.
func frame(ctx context.Context, src string) (image.Image, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("%w: video needs ffmpeg", ErrNoDecoder)
	}
	scale := "scale='min(" + strconv.Itoa(Size) + ",iw)':-2"
	cmd := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-i", src,
		"-frames:v", "1", "-vf", scale, "-f", "image2pipe", "-vcodec", "mjpeg", "-")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}
	return img, nil
}

// fit scales img so its long edge is Size. ApproxBiLinear, not CatmullRom:
// on a 12 MP photo it takes 3 ms instead of 800 (measured 2026-10-03), and at
// 256 pixels the difference does not show.
func fit(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= Size && h <= Size {
		return img
	}
	if w >= h {
		w, h = Size, max(1, h*Size/w)
	} else {
		w, h = max(1, w*Size/h), Size
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// upright turns img as the EXIF orientation says.
func upright(img image.Image, orientation int) image.Image {
	if orientation <= 1 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := orientation >= 5
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if swap {
		out = image.NewRGBA(image.Rect(0, 0, h, w))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch orientation {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			out.Set(nx, ny, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}
