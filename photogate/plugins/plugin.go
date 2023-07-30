package plugins

import (
	"reflect"
	"strings"

	"github.com/fogleman/gg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
)

type Plugin interface {
	Type() string
	Configure() error
	Apply(*gg.Context) error
}

type bindablePlugin interface {
	// return a new instance with new field values or self if not change
	Bind(BindValues) (Plugin, error)
}

type BindValues map[string]interface{}

func (v BindValues) Get(key string) interface{} {
	return v[key]
}

func (v BindValues) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

type BindMapping struct {
	Binding map[string]string
	binded  bool
}

// convert all strings to lowercase
func (m BindMapping) normalize() {
	for k, v := range m.Binding {
		m.Binding[strings.ToLower(k)] = strings.ToLower(v)
	}
}

var registeredPlugins map[string]reflect.Type = make(map[string]reflect.Type)

func register(p Plugin) {
	t := p.Type()
	if _, ok := registeredPlugins[t]; ok {
		log.Fatal().Msgf("type %s already registered", t)
	}
	registeredPlugins[t] = reflect.TypeOf(p)
}

func newInstanceOf(typ string) Plugin {
	v, ok := registeredPlugins[typ]
	if !ok {
		return nil
	}
	return reflect.New(v.Elem()).Interface().(Plugin)
}
