package plugins

import (
	"fmt"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/spf13/cast"
	"gitlab.sendo.vn/system/photogate/utils"
	"golang.org/x/image/font"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	printer = message.NewPrinter(language.Vietnamese)
)

func init() {
	register(&TextPlugin{})
}

type TextPlugin struct {
	BindMapping
	Text           string
	Price          int64
	PromotionPrice int64
	DecimalPrice   int64
	IntPrice       int64

	font         font.Face
	X            float64
	Y            float64
	Color        string
	FontUri      string
	FontSize     float64
	StrikeFull   bool
	DrawWrapped  bool
	IsDec        bool
	IsInt        bool
	IsFromSfcsdm bool
	LineSpacing  float64
	TextWidth    float64
	MaxCharacter int64
}

func (TextPlugin) Type() string {
	return "text"
}

func (tp *TextPlugin) _configure() error {
	if tp.FontUri == "" {
		return fmt.Errorf("field fontUri is required")
	}
	if tp.FontSize == 0 {
		return fmt.Errorf("field fontSize is required")
	}

	font, err := gg.LoadFontFace(tp.FontUri, tp.FontSize)
	if err != nil {
		return fmt.Errorf("load font face error %s", tp.FontUri)
	}

	tp.font = font
	return nil
}

func (p *TextPlugin) Configure() error {
	p.BindMapping.normalize()

	return p._configure()
}

func (p TextPlugin) Apply(dc *gg.Context) error {
	dc.SetFontFace(p.font)
	dc.SetHexColor(p.Color)

	// str := "Thanh Trà Việt Nam Cành Lá, túi lưới 1 kg +/-50gr"
	if p.DrawWrapped {
		ax := p.X / float64(dc.Width())
		ay := p.Y / float64(dc.Height())
		txt := utils.Ellipsis(p.Text, int(p.MaxCharacter))
		dc.DrawStringWrapped(txt, p.X, p.Y, ax, ay, p.TextWidth, p.LineSpacing, gg.AlignLeft)
		// dc.DrawStringWrapped(txt, p.X, p.Y, 0.4, 0.22, p.TextWidth, p.LineSpacing, gg.AlignLeft)
	} else {
		if p.IsDec {
			x := p.X
			intPrice := printer.Sprintf("%d.", p.IntPrice)
			w, _ := dc.MeasureString(intPrice)
			x += float64(w) * 1.35

			decimalPriceStr := cast.ToString(p.DecimalPrice)
			for len(decimalPriceStr) < 3 {
				decimalPriceStr = "0" + decimalPriceStr
			}
			decimalPrice := printer.Sprintf("%sđ", decimalPriceStr)
			dc.DrawString(decimalPrice, x, p.Y)
		} else if p.IsInt {
			intPrice := printer.Sprintf("%d.", p.IntPrice)
			dc.DrawString(intPrice, p.X, p.Y)
		} else if p.PromotionPrice > 0 && !p.StrikeFull {
			promotionPrice := printer.Sprintf("%dđ", p.PromotionPrice)
			dc.DrawString(promotionPrice, p.X, p.Y)
		} else if p.Price > p.PromotionPrice && p.PromotionPrice > 0 && p.StrikeFull && p.IsFromSfcsdm {
			x := p.X
			intPrice := printer.Sprintf("%d.", p.IntPrice)
			w, h := dc.MeasureString(intPrice)
			x += float64(w) * 1.93

			price := printer.Sprintf("%dđ", p.Price)
			w, h = dc.MeasureString(price)
			dc.DrawLine(x, p.Y-h*0.25, x+float64(w), p.Y-h*0.25)
			dc.Stroke()
			dc.DrawString(price, x, p.Y)
		} else if p.Price > p.PromotionPrice && p.PromotionPrice > 0 && p.StrikeFull {
			// Draw price
			x := p.X
			pp := printer.Sprintf("%dđ", p.PromotionPrice)
			w, h := dc.MeasureString(pp)
			x += float64(w)

			// Draw line
			price := printer.Sprintf("%dđ", p.Price)
			w, h = dc.MeasureString(price)
			dc.DrawLine(x, p.Y-h*0.25, x+float64(w), p.Y-h*0.25)
			dc.Stroke()

			dc.DrawString(price, x, p.Y)
		}
	}

	return nil
}

// return a new instance with new text
// return self if not change
func (p *TextPlugin) Bind(values BindValues) (Plugin, error) {
	newP := *p

	changed := false
	for field, key := range p.Binding {
		switch field {
		case "text":
			newP.Text = values.GetString(key)
			changed = true
		case "price":
			val := values.GetString(key)
			pr, _ := strconv.ParseInt(val, 10, 64)
			newP.Price = pr
			changed = true
		case "promotion_price":
			val := values.GetString(key)
			pr, _ := strconv.ParseInt(val, 10, 64)
			newP.PromotionPrice = pr
			changed = true
		case "decimal_price":
			val := values.GetString(key)
			pr, _ := strconv.ParseInt(val, 10, 64)
			newP.DecimalPrice = pr % 1000
			changed = true
		case "int_price":
			val := values.GetString(key)
			pr, _ := strconv.ParseInt(val, 10, 64)
			newP.IntPrice = pr / 1000
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
