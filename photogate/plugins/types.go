package plugins

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "github.com/chai2010/webp"
	_ "github.com/jdeng/goheif"
)

type FPoint struct {
	X, Y float64
}

func (r FPoint) Transform(w, h int) image.Point {
	return image.Point{
		int(r.X * float64(w)),
		int(r.Y * float64(h)),
	}
}

// for coordinate position to parent
type FRectangle struct {
	Left   float64
	Top    float64
	Right  float64
	Bottom float64
}

func (r FRectangle) Transform(w, h int) image.Rectangle {
	return image.Rect(
		int(r.Left*float64(w)),
		int(r.Top*float64(h)),
		int(r.Right*float64(w)),
		int(r.Bottom*float64(h)),
	)
}
