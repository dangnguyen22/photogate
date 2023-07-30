package imghelper

import (
	"image"

	"github.com/disintegration/imaging"
)

func ResizeInner(src image.Image, w, h int) image.Image {
	imgWHRate := float64(src.Bounds().Dx()) / float64(src.Bounds().Dy())
	if w == 0 || h == 0 {
		return imaging.Resize(src, w, h, imaging.Lanczos)
	}
	if (float64(w) / float64(h)) > imgWHRate {
		return imaging.Resize(src, 0, h, imaging.Lanczos)
	} else {
		return imaging.Resize(src, w, 0, imaging.Lanczos)
	}
}

func ResizeOuter(src image.Image, w, h int) image.Image {
	imgWHRate := float64(src.Bounds().Dx()) / float64(src.Bounds().Dy())
	if w == 0 || h == 0 {
		return imaging.Resize(src, w, h, imaging.Lanczos)
	}
	if (float64(w) / float64(h)) < imgWHRate {
		return imaging.Resize(src, 0, h, imaging.Lanczos)
	} else {
		return imaging.Resize(src, w, 0, imaging.Lanczos)
	}
}

func ResizeStretch(src image.Image, w, h int) image.Image {
	return imaging.Resize(src, w, h, imaging.Lanczos)
}
