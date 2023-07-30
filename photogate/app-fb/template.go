package appfb

import (
	"image/color"
	"io/fs"
	"regexp"
	"strings"

	"github.com/golang/freetype/truetype"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gitlab.sendo.vn/system/photogate/utils"
	"golang.org/x/image/font"
	"gopkg.in/yaml.v3"
)

var errInvalidFormat = errors.New("invalid color format")

func parseHexColor(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errInvalidFormat
	}
	return
}

type TextConfig struct {
	Top   float32 `yaml:",omitempty"`
	Right float32 `yaml:",omitempty"`
	// make right is center point
	VerticalCenter bool `yaml:"verticalCenter,omitempty"`

	Height  float64 `yaml:",omitempty"`
	FontURI string  `yaml:"fontUri,omitempty"`
	Color   string  `yaml:",omitempty"`

	// strikethrough by percent of text height
	StrikeThrough float64 `yaml:"strikeThrough,omitempty"`
	StrikeFull    bool    `yaml:",omitempty"`
	StrikePos     float64 `yaml:",omitempty"`

	_color color.RGBA
	_font  *truetype.Font
}

func (tc *TextConfig) parse(reference *TextConfig) error {
	var err error
	if tc.Color == "" {
		tc._color = color.RGBA{255, 255, 255, 255}
	} else {
		tc._color, err = parseHexColor(tc.Color)
		if err != nil {
			return err
		}
	}

	if tc.FontURI == "" {
		if reference == nil {
			return errors.New("FontURI must be valid")
		} else {
			tc._font = reference._font
		}
	} else {
		var buf []byte
		buf, err = utils.SimpleGetFile(tc.FontURI)
		if err != nil {
			return errors.WithMessagef(err, `load font "%s"`, tc.FontURI)
		}

		tc._font, err = truetype.Parse(buf)
		if err != nil {
			return err
		}
	}

	if tc.StrikeThrough > 1 {
		tc.StrikeThrough = 1
	} else if tc.StrikeThrough < 0 {
		tc.StrikeThrough = 0
	}

	return nil
}

const textDPI = 96

func (tc *TextConfig) newFace(imageHeight int) font.Face {
	fsize := (float64(imageHeight) * tc.Height) * 72 / textDPI
	return truetype.NewFace(tc._font, &truetype.Options{
		Size: fsize,
		DPI:  textDPI,
	})
}

type ImageTemplateConfig struct {
	FrameURI string `yaml:"frameURI,omitempty"`

	PromoFrameURI string `yaml:"promoFrameURI,omitempty"`

	PriceOnly  TextConfig `yaml:"priceOnly,omitempty"`
	PriceOrig  TextConfig `yaml:"priceOrig,omitempty"`
	PricePromo TextConfig `yaml:"pricePromo,omitempty"`
}

func loadImageTemplateConfig(s string) (*ImageTemplateConfig, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}

	cc := &ImageTemplateConfig{}

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		TagName: "yaml",
		Result:  cc,
	})
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(m); err != nil {
		return nil, err
	}

	return cc, nil
}

var genericTemplateRx = regexp.MustCompile(`\.yaml$`)

func loadTemplates(log zerolog.Logger, staticFs fs.FS, root string) (map[string]*ImageTemplate, error) {
	tmpls := map[string]*ImageTemplate{}
	err := utils.ScanFileMatch(staticFs, root, genericTemplateRx, func(fname, path string, b []byte) error {
		name := strings.TrimSuffix(fname, ".yaml")
		log.Info().Msgf(`load fb template "%s"`, path)

		itc, err := loadImageTemplateConfig(string(b))
		if err != nil {
			return err
		}

		t, err := NewImageTemplate(itc)
		if err != nil {
			return err
		}

		tmpls[name] = t

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tmpls, err
}
