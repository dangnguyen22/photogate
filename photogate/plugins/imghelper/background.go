package imghelper

import (
	"image/color"

	"github.com/fogleman/gg"
)

func InitDrawingContext(w, h int, c color.Color) *gg.Context {
	dc := gg.NewContext(w, h)
	if c != nil && c != color.Transparent {
		dc.SetColor(c)
		dc.Clear()
	}
	return dc
}
