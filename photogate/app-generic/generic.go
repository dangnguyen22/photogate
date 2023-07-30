package appgeneric

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.sendo.vn/system/photogate/downloader"
	"gitlab.sendo.vn/system/photogate/logger"
	"gitlab.sendo.vn/system/photogate/plugins"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
)

var (
	TemplateName = "template"
)

var values plugins.BindValues

var opsDurationProcessed = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "photogate_generic_response_duration_time",
	Help: "Duration of generic generate image requests",
}, []string{TemplateName})

func init() {
	viper.SetDefault("generic.loglevel", "debug")
}

func init() {
	prometheus.Register(opsDurationProcessed)
}

type genericService struct {
	mr *mux.Router
	ir *mux.Router

	tmpls map[string]*template

	upstream string

	log zerolog.Logger
}

func NewGenericService(media3 string, templateFs fs.FS) (*genericService, error) {
	mr := mux.NewRouter()
	ir := mux.NewRouter()

	log := logger.NamedLogger("gapp").Level(logger.GetLogLevel("generic.loglevel"))

	tmpls, err := loadTemplates(log, templateFs, "generic-templates")
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(media3, "/") {
		media3 += "/"
	}

	s := &genericService{
		mr:       mr,
		ir:       ir,
		tmpls:    tmpls,
		upstream: media3,
	}

	singleItemSR := mr.PathPrefix("/{template}").Subrouter()
	singleItemSR.Path("/{source:.*}").Methods(http.MethodGet).HandlerFunc(s.handleImage)
	singleItemSR.Use(s.mwGetSingleItem)

	doubleItemSR := mr.PathPrefix("/{template}").Subrouter()
	doubleItemSR.Methods(http.MethodGet).HandlerFunc(s.handleImage)
	doubleItemSR.Use(s.mwGetDoubleItem)

	return s, nil
}

func (svc *genericService) MainHandler() http.Handler {
	return svc.mr
}

func (svc *genericService) InternalHandler() http.Handler {
	return svc.ir
}

func (svc *genericService) handleImage(w http.ResponseWriter, r *http.Request) {
	template := values.GetString("template")
	// start := time.Now()
	timer := prometheus.NewTimer(opsDurationProcessed.With(prometheus.Labels{"template": template}))
	defer timer.ObserveDuration()

	tmpl, ok := svc.tmpls[template]
	if !ok {
		log.Error().Msgf("template %s not found", template)
		w.Header().Add("content-type", "image/png")
		w.WriteHeader(400)
		w.Write(imghelper.Empty1x1_PNG)
		return
	}

	img, err := tmpl.Render(values, 0)
	if err != nil {
		w.Header().Add("content-type", "image/png")
		if err2, ok := err.(*downloader.DownloadError); ok {
			w.WriteHeader(err2.Code)
		} else {
			w.WriteHeader(500)
		}
		w.Write(imghelper.Empty1x1_PNG)
		return
	}

	w.Header().Add("content-type", "image/jpeg")
	w.Write(imghelper.Img2jpegBuf(img))
}

func (svc *genericService) mwGetSingleItem(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		template := vars["template"]
		source := vars["source"]
		productName := r.URL.Query().Get("product_name")
		price := r.URL.Query().Get("price")
		promotionPrice := r.URL.Query().Get("promotion_price")
		values = plugins.BindValues{
			"template":        template,
			"source":          svc.upstream + source,
			"product_name":    productName,
			"price":           price,
			"promotion_price": promotionPrice,
		}

		next.ServeHTTP(w, r)
	})
}

func (svc *genericService) mwGetDoubleItem(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		template := vars["template"]
		img_source1 := r.URL.Query().Get("img_source1")
		price1 := r.URL.Query().Get("price1")
		promotionPrice1 := r.URL.Query().Get("promotion_price1")
		img_source2 := r.URL.Query().Get("img_source2")
		price2 := r.URL.Query().Get("price2")
		promotionPrice2 := r.URL.Query().Get("promotion_price2")
		values = plugins.BindValues{
			"template":         template,
			"img_source1":      svc.upstream + img_source1,
			"price1":           price1,
			"promotion_price1": promotionPrice1,
			"img_source2":      svc.upstream + img_source2,
			"price2":           price2,
			"promotion_price2": promotionPrice2,
		}

		next.ServeHTTP(w, r)
	})
}
