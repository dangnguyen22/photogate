package plugins

import (
	"errors"
	"fmt"
	"image"

	"github.com/fogleman/gg"
	"github.com/rs/zerolog/log"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
)

func init() {
	register(&ImagePlugin{})
}

type H_ALIGN string
type V_ALIGN string
type IMAGE_RESIZE_MODE string
type IMAGE_TYPE string

const (
	HALIGN_LEFT         H_ALIGN = "left"
	HALIGN_LEFT_MARGIN  H_ALIGN = "left_margin"
	HALIGN_RIGHT_MARGIN H_ALIGN = "right_margin"
	HALIGN_CENTER       H_ALIGN = "center"
	HALIGN_RIGHT        H_ALIGN = "right"
	VALIGN_TOP          V_ALIGN = "top"
	VALIGN_MIDDLE       V_ALIGN = "middle"
	VALIGN_BOTTOM       V_ALIGN = "bottom"

	MODE_CLIP    IMAGE_RESIZE_MODE = "clip"
	MODE_CROP    IMAGE_RESIZE_MODE = "crop"
	MODE_STRETCH IMAGE_RESIZE_MODE = "stretch"

	IMAGE_TYPE_PRODUCT IMAGE_TYPE = "product"
)

type ImagePlugin struct {
	BindMapping

	Image   string
	ImgType IMAGE_TYPE
	Width   int
	Height  int
	X       int
	Y       int
	Rect    FRectangle
	Mode    IMAGE_RESIZE_MODE
	HAlign  H_ALIGN
	VAlign  V_ALIGN

	_img image.Image
}

func (ImagePlugin) Type() string {
	return "image"
}

func (p *ImagePlugin) _configure() error {
	var err error
	if p.Image == "" {
		return fmt.Errorf(`field image is required`)
	}
	p._img, err = imghelper.LoadImage(p.Image)

	if p.Rect.Right == 0 {
		p.Rect.Right = 1
	}
	if p.Rect.Bottom == 0 {
		p.Rect.Bottom = 1
	}

	switch p.HAlign {
	case HALIGN_LEFT, HALIGN_LEFT_MARGIN, HALIGN_CENTER, HALIGN_RIGHT, HALIGN_RIGHT_MARGIN:
		// pass
	case "":
		p.HAlign = HALIGN_CENTER
	default:
		return errors.New("invalid halign")
	}

	switch p.VAlign {
	case VALIGN_TOP, VALIGN_MIDDLE, VALIGN_BOTTOM:
		// pass
	case "":
		p.VAlign = VALIGN_MIDDLE
	default:
		return errors.New("invalid valign")
	}

	switch p.Mode {
	case MODE_CLIP, MODE_CROP, MODE_STRETCH:
		// pass
	case "":
		p.Mode = MODE_CLIP
	default:
		return errors.New("invalid valign")
	}

	return err
}

func (p *ImagePlugin) Configure() error {
	p.BindMapping.normalize()

	if len(p.Binding) > 0 {
		return nil
	}

	return p._configure()
}

func _getResizer(mode IMAGE_RESIZE_MODE) func(image.Image, int, int) image.Image {
	switch mode {
	case MODE_CLIP:
		return imghelper.ResizeInner
	case MODE_CROP:
		return imghelper.ResizeOuter
	case MODE_STRETCH:
		return imghelper.ResizeStretch
	default:
		log.Fatal().Msg("invalid mode")
		return nil
	}
}

func (p ImagePlugin) _get_halign(align H_ALIGN, r image.Rectangle) (int, float64) {
	var (
		ax float64
		x  int
	)
	switch align {
	case HALIGN_CENTER:
		ax = 0.5
		x = (r.Max.X + r.Min.X) / 2
	case HALIGN_RIGHT:
		ax = 1
		x = r.Max.X
	case HALIGN_LEFT_MARGIN:
		ax = float64(p.X / r.Max.X)
		x = p.X
	case HALIGN_RIGHT_MARGIN:
		ax = 1 - float64(p.X/r.Max.X)
		x = r.Max.X - p.X
	default:
	}
	return x, ax
}

func (p ImagePlugin) _get_valign(align V_ALIGN, r image.Rectangle) (int, float64) {
	var (
		ay float64
		y  int
	)
	switch align {
	case VALIGN_MIDDLE:
		ay = 0.5
		y = (r.Max.Y+r.Min.Y)/2 + p.Y
	case VALIGN_BOTTOM:
		ay = 1
		y = r.Max.Y
	default:
	}
	return y, ay
}

func (p ImagePlugin) Apply(dc *gg.Context) error {
	r := p.Rect.Transform(dc.Width(), dc.Height())

	img := p._img

	isCorrectSize := img.Bounds().Dx() == r.Dx() && img.Bounds().Dy() == r.Dy()
	if !isCorrectSize && p.ImgType == IMAGE_TYPE_PRODUCT {
		img = _getResizer(p.Mode)(img, p.Width, p.Height) // Resize by w, h configured
	} else {
		img = _getResizer(p.Mode)(img, r.Dx(), r.Dy()) // Resize by rect frame
	}

	x, ax := p._get_halign(p.HAlign, r)
	y, ay := p._get_valign(p.VAlign, r)

	dc.MoveTo(float64(r.Min.X), float64(r.Min.Y))
	dc.LineTo(float64(r.Max.X), float64(r.Min.Y))
	dc.LineTo(float64(r.Max.X), float64(r.Max.Y))
	dc.LineTo(float64(r.Min.X), float64(r.Max.Y))
	dc.Clip()
	dc.DrawImageAnchored(img, x, y, ax, ay)

	dc.ResetClip()

	return nil
}

func (p *ImagePlugin) Bind(values BindValues) (Plugin, error) {
	newP := *p

	changed := false
	for field, key := range p.Binding {
		switch field {
		case "image":
			newP.Image = values.GetString(key)
			changed = true
		}
	}

	if !changed {
		return p, nil
	}

	err := newP._configure()
	if err != nil {
		return nil, err
	}

	return &newP, nil
}
