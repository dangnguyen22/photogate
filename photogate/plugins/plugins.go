package plugins

import (
	"fmt"

	"github.com/fogleman/gg"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Plugins []Plugin

func NewStructDecoder(out interface{}) (*mapstructure.Decoder, error) {
	return mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		TagName: "yaml",
		Squash:  true,
		Result:  out,
	})
}

func NewPluginsFromConfig(configs []map[string]interface{}) (Plugins, error) {
	plugins := make(Plugins, 0, len(configs))
	for i, m := range configs {
		t, ok := m["type"].(string)
		if !ok {
			return nil, fmt.Errorf("plugins at index %d invalid", i)
		}

		p := newInstanceOf(t)
		if p == nil {
			return nil, fmt.Errorf("plugins type %s not found", t)
		}

		dec, err := NewStructDecoder(p)
		if err != nil {
			return nil, err
		}

		if err = dec.Decode(m); err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}

	return plugins, nil
}

func (ps Plugins) Configure() error {
	for i, l := range ps {
		err := l.Configure()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(`configure plugin #%d (%s)`, i, l.Type()))
		}
	}
	return nil
}

func (ps Plugins) Execute(dc *gg.Context) error {
	for _, p := range ps {
		if err := p.Apply(dc); err != nil {
			return err
		}
	}
	return nil
}

func (ps Plugins) Bind(values BindValues) (Plugins, error) {
	ps2 := make(Plugins, 0, len(ps))

	for i, p := range ps {
		if dp, ok := p.(bindablePlugin); ok {
			log.Debug().
				Interface("values", values).
				Int("index", i).
				Str("type", p.Type()).
				Msg("bind")
			binded, err := dp.Bind(values)
			if err != nil {
				return nil, err
			}
			ps2 = append(ps2, binded)
		} else {
			ps2 = append(ps2, p)
		}
	}

	return ps2, nil
}
