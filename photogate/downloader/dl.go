// downloader service
package downloader

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.sendo.vn/system/photogate/logger"
)

var (
	TagName = "tag"
)

var imageDownloadDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "photogate_image_download_duration_time",
	Help: "Duration download image",
}, []string{TagName})

func init() {
	viper.SetDefault("downloader.concurrent", 2*runtime.NumCPU())
	viper.SetDefault("downloader.loglevel", "debug")
}

func init() {
	prometheus.Register(imageDownloadDuration)
}

var dlsvc *downloadService

func Init() {
	if dlsvc != nil {
		log.Fatal().Msg("downloader already init")
	}
	dlsvc = newDownloadService(viper.GetInt("downloader.concurrent"))
}

type DownloadError struct {
	Code int
	Body []byte
}

func (e *DownloadError) Error() string {
	return fmt.Sprintf("error %d", e.Code)
}

type downloadService struct {
	client http.Client

	cond *sync.Cond

	dlCurrent int
	dlMax     int

	log zerolog.Logger
}

func newDownloadService(max int) *downloadService {
	return &downloadService{
		cond:  sync.NewCond(&sync.Mutex{}),
		dlMax: max,
		client: http.Client{
			Timeout: time.Second * 10,
		},
		log: logger.NamedLogger("downloader").Level(logger.GetLogLevel("downloader.loglevel")),
	}
}

func (ds *downloadService) Download(uri string, tag string) ([]byte, error) {
	start := time.Now()

	ds.cond.L.Lock()
	for ds.dlCurrent >= ds.dlMax {
		ds.cond.Wait()
	}
	ds.dlCurrent++
	ds.cond.L.Unlock()

	defer func() {
		ds.cond.L.Lock()
		ds.dlCurrent--
		ds.cond.L.Unlock()
		ds.cond.Signal()
	}()

	waitTime := time.Since(start)
	if waitTime < time.Millisecond {
		waitTime = 0
	}
	dlStart := time.Now()

	var err error
	var code int

	defer func() {
		lg := ds.log.Info().
			Str("tag", tag).
			Str("target", uri).
			Dur("wait", waitTime).
			Dur("dur", time.Since(dlStart)).
			Int("code", code)
		if err != nil {
			lg.Err(err).Msg("download")
		} else {
			lg.Msg("downloaded")
		}
	}()

	resp, err := ds.client.Get(uri)
	if err != nil {
		return nil, err
	}
	code = resp.StatusCode

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)

		err = &DownloadError{Code: resp.StatusCode, Body: b}
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func Download(uri string, tag string) (b []byte, err error) {
	timer := prometheus.NewTimer(imageDownloadDuration.With(prometheus.Labels{"tag": tag}))
	defer timer.ObserveDuration()
	return dlsvc.Download(uri, tag)
}
