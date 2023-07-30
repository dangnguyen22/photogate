package appgeneric

import (
	"errors"
	"image"
	"image/color"
	"io/fs"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.sendo.vn/system/photogate/plugins"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
	"gitlab.sendo.vn/system/photogate/utils"
	"gopkg.in/yaml.v3"
)

type template struct {
	// possible widths, default width = widths[0]
	AllWidths []int
	// height = width / ratio
	WidthHeightRatio float64

	BackgroundColor string
	_bgColor        color.Color

	Plugins  []map[string]interface{}
	_plugins plugins.Plugins
}

func intsIndex(arr []int, x int) int {
	for i, v := range arr {
		if x == v {
			return i
		}
	}
	return -1
}

func (tm *template) Render(values plugins.BindValues, width int) (image.Image, error) {
	if intsIndex(tm.AllWidths, width) < 0 {
		width = tm.AllWidths[0]
	}
	height := int(float64(width) / tm.WidthHeightRatio)

	var err error

	ps, err := tm._plugins.Bind(values)
	if err != nil {
		log.Error().Err(err).Msg("bind")
		return nil, err
	}

	dc := imghelper.InitDrawingContext(width, height, tm._bgColor)
	err = ps.Execute(dc)
	if err != nil {
		log.Error().Err(err).Msg("execute")
		return nil, err
	}

	return dc.Image(), nil
}

func loadTemplate(name string, b []byte) (*template, error) {
	var m map[string]interface{}

	err := yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	c := &template{}
	dec, err := plugins.NewStructDecoder(c)
	if err != nil {
		return nil, err
	}
	if err = dec.Decode(m); err != nil {
		return nil, err
	}

	if len(c.AllWidths) < 1 {
		return nil, errors.New("allWidths is required")
	}

	if c.WidthHeightRatio <= 0 {
		c.WidthHeightRatio = 1
	}

	if c.BackgroundColor != "" {
		c._bgColor = imghelper.ParseColor(c.BackgroundColor)
	}

	c._plugins, err = plugins.NewPluginsFromConfig(c.Plugins)
	if err != nil {
		return nil, err
	}

	err = c._plugins.Configure()
	if err != nil {
		return nil, err
	}

	return c, nil
}

var genericTemplateRx = regexp.MustCompile(`\.yaml$`)

func loadTemplates(log zerolog.Logger, static fs.FS, root string) (map[string]*template, error) {
	tmpls := map[string]*template{}
	err := utils.ScanFileMatch(static, root, genericTemplateRx, func(fname, path string, b []byte) error {
		name := strings.TrimSuffix(fname, ".yaml")

		log.Info().Msgf(`load generic template "%s"`, path)
		t, err := loadTemplate(name, b)
		if err != nil {
			return err
		}

		tmpls[name] = t
		return nil
	})

	if err != nil {
		return nil, err
	}

	return tmpls, nil
}
