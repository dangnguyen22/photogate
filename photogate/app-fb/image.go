package appfb

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"sync"

	_ "image/gif"

	_ "github.com/chai2010/webp"
	_ "github.com/jdeng/goheif"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
	"gitlab.sendo.vn/system/photogate/utils"

	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	printer = message.NewPrinter(language.Vietnamese)
)

type ImageTemplate struct {
	mu  *sync.Mutex
	cfg *ImageTemplateConfig

	origFrame      image.Image
	promoOrigFrame image.Image
	// frame by size
	frames map[string]image.Image
}

func NewImageTemplate(cfg *ImageTemplateConfig) (*ImageTemplate, error) {
	var origFrame, promoOrigFrame image.Image

	b, err := utils.SimpleGetFile(cfg.FrameURI)
	if err != nil {
		return nil, err
	}
	origFrame, _, err = image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	if cfg.PromoFrameURI != "" {
		b, err := utils.SimpleGetFile(cfg.PromoFrameURI)
		if err != nil {
			return nil, err
		}
		promoOrigFrame, _, err = image.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
	}

	if err := cfg.PriceOnly.parse(nil); err != nil {
		return nil, errors.WithMessage(err, "PriceOnly")
	}
	if err := cfg.PriceOrig.parse(&cfg.PriceOnly); err != nil {
		return nil, errors.WithMessage(err, "PriceOrig")
	}
	if err := cfg.PricePromo.parse(&cfg.PriceOnly); err != nil {
		return nil, errors.WithMessage(err, "PricePromo")
	}

	return &ImageTemplate{
		mu:             &sync.Mutex{},
		cfg:            cfg,
		origFrame:      origFrame,
		promoOrigFrame: promoOrigFrame,
		frames:         make(map[string]image.Image),
	}, nil
}

func (it *ImageTemplate) getFrameBySize(s image.Point, isPromo bool) image.Image {
	k := s.String()

	if isPromo && it.promoOrigFrame != nil {
		k = "promo_" + k
	}

	it.mu.Lock()
	defer it.mu.Unlock()
	img, ok := it.frames[k]
	if !ok {
		it.mu.Unlock()
		fr := it.origFrame
		if isPromo && it.promoOrigFrame != nil {
			fr = it.promoOrigFrame
		}
		img = imaging.Resize(fr, s.X, s.Y, imaging.Lanczos)

		it.mu.Lock()
		it.frames[k] = img
	}

	return img
}

func (it *ImageTemplate) fillRect(dst draw.Image, rect image.Rectangle, color color.Color) {
	for y := rect.Min.Y; y <= rect.Max.Y; y++ {
		for x := rect.Min.X; x <= rect.Max.X; x++ {
			dst.Set(x, y, color)
		}
	}
}

func (it *ImageTemplate) _strikeThrough(dst draw.Image, tc *TextConfig, face font.Face, rect image.Rectangle) {
	if tc.StrikeThrough <= 0 {
		return
	}

	rightOffset := 0
	if !tc.StrikeFull {
		vndWidth := font.MeasureString(face, " đ").Floor()
		rightOffset = -vndWidth
	}

	height := rect.Dy() + 1
	lineHeight := int(math.Ceil(tc.StrikeThrough * float64(height)))

	baseline := int(float64(rect.Min.Y) + float64(height)*tc.StrikePos)
	baseline = baseline - lineHeight/2

	rect2 := image.Rect(
		rect.Min.X,
		baseline,
		rect.Max.X+rightOffset,
		baseline+lineHeight-1,
	)
	it.fillRect(dst, rect2, tc._color)
}

func (it *ImageTemplate) drawPricing(dst draw.Image, anchorPoint fixed.Point26_6, price int, tc *TextConfig) {
	face := tc.newFace(dst.Bounds().Dy())
	defer face.Close()

	txt := printer.Sprintf("%d đ", price)
	bounds, _ := font.BoundString(face, txt)

	// compute Point26_6
	txtWidth := bounds.Max.X - bounds.Min.X
	txtHeight := bounds.Max.Y - bounds.Min.Y
	fRight := anchorPoint.X
	fTop := anchorPoint.Y

	// drawer baseline
	baseLeft := fRight - txtWidth
	baseBottom := fTop + txtHeight

	if tc.VerticalCenter {
		baseLeft = fRight - txtWidth/2
		fRight += txtWidth / 2
	}

	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(tc._color),
		Face: face,
		Dot:  fixed.Point26_6{X: baseLeft, Y: baseBottom},
	}
	d.DrawString(txt)

	rect := image.Rect(baseLeft.Floor(), fTop.Floor(), fRight.Ceil(), baseBottom.Ceil())
	it._strikeThrough(dst, tc, face, rect)
}

func (it *ImageTemplate) Generate(buf []byte, price, promotionPrice int) ([]byte, error) {
	return it.GenerateReader(bytes.NewReader(buf), price, promotionPrice)
}

func (it *ImageTemplate) GenerateReader(rd io.Reader, price, promotionPrice int) ([]byte, error) {
	src, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}
	return it.GenerateFromImage(src, price, promotionPrice), nil
}

func (it *ImageTemplate) _calcAnchorPoint(tc *TextConfig, imageSize image.Point) fixed.Point26_6 {
	right := imageSize.X - int(tc.Right*float32(imageSize.X))
	top := int(tc.Top * float32(imageSize.Y))

	return fixed.P(right, top)
}

func (it *ImageTemplate) GenerateFromImage(src image.Image, price, promotionPrice int) []byte {
	imageSize := src.Bounds().Size()

	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, image.Point{}, draw.Src)

	frame := it.getFrameBySize(imageSize, price != promotionPrice)
	draw.Draw(dst, dst.Bounds(), frame, image.Point{}, draw.Over)

	if price == promotionPrice {
		tc := &it.cfg.PriceOnly
		p := it._calcAnchorPoint(tc, imageSize)
		it.drawPricing(dst, p, price, tc)
	} else {
		tc := &it.cfg.PriceOrig
		p := it._calcAnchorPoint(tc, imageSize)
		it.drawPricing(dst, p, price, tc)

		tc = &it.cfg.PricePromo
		p = it._calcAnchorPoint(tc, imageSize)
		it.drawPricing(dst, p, promotionPrice, tc)
	}

	return imghelper.Img2jpegBuf(dst)
}

func (it *ImageTemplate) getFrameBySizeNoPrice(s image.Point) image.Image {
	k := s.String()

	it.mu.Lock()	
	defer it.mu.Unlock()
	img, ok := it.frames[k]
	if !ok {
		it.mu.Unlock()
		fr := it.origFrame
		img = imaging.Resize(fr, s.X, s.Y, imaging.Lanczos)

		it.mu.Lock()
		it.frames[k] = img
	}
	return img
}

func (it *ImageTemplate) GenerateFromImageNotPrice(src image.Image) []byte {
	imageSize := src.Bounds().Size()

	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, image.Point{}, draw.Src)

	frame := it.getFrameBySizeNoPrice(imageSize)
	draw.Draw(dst, dst.Bounds(), frame, image.Point{}, draw.Over)
	return imghelper.Img2jpegBuf(dst)
}