package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// setup default logger
func init() {
	log.Logger = NamedLogger("main")
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite
)

func colorize(s interface{}, c int, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func GetLogLevel(key string) zerolog.Level {
	s := viper.GetString(key)

	lv, err := zerolog.ParseLevel(s)
	if err != nil {
		log.Error().Str("key", key).Msgf("invalid loglevel %s", s)
		return zerolog.InfoLevel
	}

	return lv
}

func NamedLogger(name string) zerolog.Logger {
	noColor := false

	name2 := colorize(name, colorCyan, noColor) + " "

	lg := log.With().
		Caller().
		Logger().
		Output(&zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: noColor,
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
			FormatCaller: func(i interface{}) string {
				s := i.(string)
				parts := strings.Split(s, "/")
				l := len(parts)
				if l > 2 {
					s = parts[l-2] + "/" + parts[l-1]
				}
				s = colorize(s, colorMagenta, noColor)
				return name2 + s
			},
		})
	return lg
}
