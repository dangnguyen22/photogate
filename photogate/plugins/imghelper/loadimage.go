package imghelper

import (
	"bytes"
	"image"

	"gitlab.sendo.vn/system/photogate/utils"
)

// load image via static or http
func LoadImage(uri string) (image.Image, error) {
	b, err := utils.SimpleGetFile(uri)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err
}
