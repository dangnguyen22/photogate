package main

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	appfb "gitlab.sendo.vn/system/photogate/app-fb"
	appgeneric "gitlab.sendo.vn/system/photogate/app-generic"
	appqr "gitlab.sendo.vn/system/photogate/app-qr"
	"gitlab.sendo.vn/system/photogate/downloader"
	"gitlab.sendo.vn/system/photogate/utils"
)

var (
	//go:embed static
	__embedFs embed.FS
	// fs without static prefix
	staticFs fs.FS
)

type handlerService interface {
	MainHandler() http.Handler
	InternalHandler() http.Handler
}

func init() {
	var err error
	staticFs, err = fs.Sub(__embedFs, "static")
	if err != nil {
		log.Fatal().Err(err).Msg("static dir")
	}

	utils.Init(staticFs)
}

type App struct {
	r    *mux.Router
	port int

	chStop chan struct{}
}

func registerService(r *mux.Router, prefix string, hs handlerService) {
	prefix = strings.TrimSuffix(prefix, "/")

	if h := hs.MainHandler(); h != nil {
		r.PathPrefix(prefix + "/").Handler(
			http.StripPrefix(prefix, h),
		)
		r.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
		})
	}

	iprefix := "/internal" + prefix
	if h := hs.InternalHandler(); h != nil {
		r.PathPrefix(iprefix + "/").Handler(
			http.StripPrefix(iprefix, h),
		)
		r.HandleFunc(iprefix, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, iprefix+"/", http.StatusMovedPermanently)
		})
	}
}

func NewApp() (*App, error) {
	r := mux.NewRouter()

	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Path("/internal/metrics").Handler(promhttp.Handler())

	downloader.Init()

	media3 := viper.GetString("media3.url")

	{
		qrSvc, err := appqr.NewQrService(staticFs)
		if err != nil {
			return nil, errors.Wrap(err, "qr-service")
		}

		registerService(r, "/qr", qrSvc)
	}

	{
		qrSvc, err := appgeneric.NewGenericService(media3, staticFs)
		if err != nil {
			return nil, errors.Wrap(err, "generic-service")
		}

		registerService(r, "/template/", qrSvc)
	}

	{
		fbApp, err := appfb.NewFbImageService(media3, staticFs)
		if err != nil {
			return nil, errors.Wrap(err, "fb-service")
		}
		registerService(r, "/fb", fbApp)
	}

	app := &App{
		r:      r,
		chStop: make(chan struct{}),
	}

	return app, nil
}

func (app *App) Start(ctx context.Context, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		app.port = -1
		return err
	}

	tcp, _ := net.ResolveTCPAddr(lis.Addr().Network(), lis.Addr().String())
	log.Info().Msgf("listening on %s", tcp.String())
	app.port = tcp.Port

	srv := &http.Server{Handler: app.r}
	go srv.Serve(lis)

	<-ctx.Done()
	srv.Shutdown(context.Background())

	return nil
}
