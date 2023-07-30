package plugins

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/skip2/go-qrcode"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
)

func init() {
	register(&QrPlugin{})
}

type QrPlugin struct {
	BindMapping

	Text     string
	Color    string
	Recovery qrcode.RecoveryLevel

	Anchor FPoint
	// (0, 1]
	Size float64

	_qr *qrcode.QRCode
}

func (QrPlugin) Type() string {
	return "qr"
}

func (p *QrPlugin) _configure() error {
	qr, err := qrcode.New(p.Text, p.Recovery)
	if err != nil {
		return err
	}
	qr.DisableBorder = true
	qr.BackgroundColor = color.Transparent
	qr.ForegroundColor = imghelper.ParseColor(p.Color)

	p._qr = qr
	return nil
}

func (p *QrPlugin) Configure() error {
	p.BindMapping.normalize()

	if p.Size > 1 || p.Size <= 0 {
		return errors.New("field size must in (0, 1]")
	}
	if p.Recovery < qrcode.Low || p.Recovery > qrcode.Highest {
		return fmt.Errorf("field recovery must >= %d && <= %d", qrcode.Low, qrcode.Highest)
	}

	if p.Text == "" {
		p.Text = "dummy"
	}

	return p._configure()
}

func (p QrPlugin) Apply(dc *gg.Context) error {
	r := p.Anchor.Transform(dc.Width(), dc.Height())
	img := p._qr.Image(int(p.Size * float64(dc.Width())))

	dc.DrawImageAnchored(img, r.X, r.Y, 0.5, 0.5)

	return nil
}

// return a new instance with new text
// return self if not change
func (p *QrPlugin) Bind(values BindValues) (Plugin, error) {
	newP := *p

	changed := false
	for field, key := range p.Binding {
		switch field {
		case "text":
			newP.Text = values.GetString(key)
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
