package embedding

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"

	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	MaxQueryImageBytes      = 10 * 1024 * 1024
	MaxNormalizedImageBytes = 3 * 1024 * 1024
)

func JPEGDataURI(content []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(content)
}

func NormalizeImage(content []byte) ([]byte, error) {
	if len(content) == 0 || len(content) > MaxQueryImageBytes {
		return nil, errors.New("图片大小不能超过 10MB")
	}
	source, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, errors.New("不支持的图片格式")
	}
	if source.Bounds().Dx() <= 0 || source.Bounds().Dy() <= 0 {
		return nil, errors.New("图片内容无效")
	}

	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	for maxEdge := max(width, height); maxEdge > 4096; maxEdge = max(width, height) {
		ratio := 4096.0 / float64(maxEdge)
		width = int(math.Max(1, float64(width)*ratio))
		height = int(math.Max(1, float64(height)*ratio))
	}

	for attempt := 0; attempt < 8; attempt++ {
		target := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(target, target.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		xdraw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
		for quality := 88; quality >= 55; quality -= 11 {
			var output bytes.Buffer
			if err := jpeg.Encode(&output, target, &jpeg.Options{Quality: quality}); err != nil {
				return nil, errors.New("图片压缩失败")
			}
			if output.Len() <= MaxNormalizedImageBytes {
				return output.Bytes(), nil
			}
		}
		width = max(1, width*3/4)
		height = max(1, height*3/4)
	}
	return nil, errors.New("图片压缩后仍超过 3MB")
}
