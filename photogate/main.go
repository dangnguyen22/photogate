package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"gitlab.sendo.vn/system/photogate/logger"
	_ "gitlab.sendo.vn/system/photogate/logger"
)

func init() {
	viper.SetConfigFile("config.yaml")
	viper.SetDefault("listen", ":8080")
	viper.SetDefault("main.loglevel", "debug")
}

func init() {
	err := viper.ReadInConfig()
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Msg(err.Error())
		} else {
			log.Fatal().Msgf(err.Error())
		}
	}

	{
		var showConfig bool
		pflag.BoolVarP(&showConfig, "show-config", "s", false, "")
		pflag.Parse()
		if showConfig {
			if b, err := yaml.Marshal(viper.AllSettings()); err != nil {
				log.Error().Err(err).Msg("marshal config")
			} else {
				log.Info().Msgf("current config: \n%s", string(b))
			}
		}
	}

	log.Logger = log.Logger.Level(logger.GetLogLevel("main.loglevel"))
}

func SetupSignals(cleanup func()) {
	notifyCtx, notifyCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer notifyCancel()
	<-notifyCtx.Done()

	cleanup()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go SetupSignals(cancel)

	app, err := NewApp()
	if err != nil {
		log.Fatal().Err(err).Msg("init photoservice")
	}
	err = app.Start(ctx, viper.GetString("listen"))
	if err == nil || err == context.Canceled {
		log.Info().Msg("graceful shutdown")
	} else {
		log.Fatal().Msg(err.Error())
	}
}

// func main() {
// 	const F = 24
// 	dc := gg.NewContext(600, 315)
// 	dc.SetRGB(1, 1, 1)
// 	dc.Clear()
// 	dc.SetRGB(63, 75, 83)
// 	if err := dc.LoadFontFace("Roboto-Bold.ttf", F); err != nil {
// 		panic(err)
// 	}
// 	str := "Thanh Trà Việt Nam Cành Lá, túi lưới 1 kg +/-50gr"
// 	// dc.DrawString(str, 239, 82)
// 	// dc.DrawStringAnchored(str, 239, 82, 0.4, 0.22)
// 	dc.DrawStringWrapped(str, 373, 82, 0.4, 0.22, 343, 1.5, gg.AlignLeft)

// 	// for r := 0; r < 256; r++ {
// 	// 	for c := 0; c < 256; c++ {
// 	// 		i := r*256 + c
// 	// 		x := float64(c*T) + T/2
// 	// 		y := float64(r*T) + T/2
// 	// 		dc.DrawStringAnchored(string(rune(i)), x, y, 0.5, 0.5)
// 	// 	}
// 	// }
// 	dc.SavePNG("out.png")
// }
