package imghelper

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
)

func Img2pngBuf(img image.Image) []byte {
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func Img2jpegBuf(img image.Image) []byte {
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 95}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
