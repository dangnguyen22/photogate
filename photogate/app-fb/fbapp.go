package appfb

import (
	"bytes"
	"encoding/json"
	"image"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gitlab.sendo.vn/system/photogate/downloader"
	"gitlab.sendo.vn/system/photogate/logger"
	"gitlab.sendo.vn/system/photogate/utils"
	"gopkg.in/yaml.v3"
)

var (
	TemplateName = "template"
)

var opsDurationProcessed = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "photogate_fb_response_duration_time",
	Help: "Duration of facebook generate image requests",
}, []string{TemplateName})

func init() {
	viper.SetDefault("fb.loglevel", "debug")
}

func init() {
	prometheus.Register(opsDurationProcessed)
}

// app render image use params in url
type fbImageService struct {
	mr *mux.Router
	ir *mux.Router

	tmpls    map[string]*ImageTemplate
	upstream string

	log zerolog.Logger
}

func NewFbImageService(media3 string, staticFs fs.FS) (*fbImageService, error) {
	mr := mux.NewRouter()
	ir := mux.NewRouter()

	log := logger.NamedLogger("fb").Level(logger.GetLogLevel("fb.loglevel"))

	tmpls, err := loadTemplates(log, staticFs, "fb-templates")
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(media3, "/") {
		media3 += "/"
	}

	ps := &fbImageService{
		mr:       mr,
		ir:       ir,
		tmpls:    tmpls,
		upstream: media3,
		log:      log,
	}

	ir.HandleFunc("/get-templates", ps.handleDebugListTemplates)
	ir.HandleFunc("/get-config", ps.handleDebugConfig)
	ir.HandleFunc("/test-template", ps.handleDebugImage)

	subFs, err := fs.Sub(staticFs, "html")
	if err != nil {
		return nil, err
	}
	ir.Methods("GET").PathPrefix("/").Handler(
		http.FileServer(http.FS(subFs)),
	)

	// ir.Handle("/", http.RedirectHandler("/debug", http.StatusTemporaryRedirect))

	// fb scdn handler
	mr.Methods("GET").Path("/get-fb-templates").HandlerFunc(ps.handleGetFBTemplates)
	mr.Methods("POST").Path("/create-fb-template").HandlerFunc(ps.handleCreateFBTemplate)
	mr.Methods("DELETE").Path("/del-fb-template/{id}").HandlerFunc(ps.handleDelFBTemplate)
	mr.Methods("GET").Path("/{source:.*}").HandlerFunc(ps.handleFbScdnImage)

	return ps, nil
}

func respondError(w http.ResponseWriter, code int, msg string) {
	respondData(w, code, map[string]string{"error": msg})
}

func respondData(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (ps *fbImageService) MainHandler() http.Handler {
	return ps.mr
}

func (ps *fbImageService) InternalHandler() http.Handler {
	return ps.ir
}

func (ps *fbImageService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps.mr.ServeHTTP(w, r)
}

func uploadFrameImage(w http.ResponseWriter, r *http.Request) (frameUrl string, err error) {
	folderName := strconv.FormatInt(utils.MakeTimestamp(), 10)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", err
	}

	defer file.Close()

	err = os.MkdirAll("static/"+folderName, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}
	dst, err := os.Create("static/" + folderName + "/frame.png")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}

	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}
	frameUrl = folderName + "/frame.png"
	return frameUrl, err
}

func (ps *fbImageService) handleDelFBTemplate(w http.ResponseWriter, r *http.Request) {

}

func (ps *fbImageService) handleCreateFBTemplate(w http.ResponseWriter, r *http.Request) {
	templateDefault := ImageTemplateConfig{
		PriceOnly: TextConfig{
			Top:            0.932,
			Right:          0.17,
			VerticalCenter: true,
			Height:         0.042,
			FontURI:        "2020-02/UTMAVO-REGULAR.TTF",
			Color:          "#fffefe",
		},
		PriceOrig: TextConfig{
			Top:            0.913,
			Right:          0.17,
			VerticalCenter: true,
			Height:         0.025,
			FontURI:        "2020-02/UTMAVO-REGULAR.TTF",
			Color:          "#ffffff",
			StrikeThrough:  0.07,
			StrikeFull:     true,
			StrikePos:      0.55,
		},
		PricePromo: TextConfig{
			Top:            0.932,
			Right:          0.17,
			VerticalCenter: true,
			Height:         0.042,
			FontURI:        "2020-02/UTMAVO-REGULAR.TTF",
			Color:          "#fffefe",
		},
	}
	frameUrl, err := uploadFrameImage(w, r)
	if err != nil {
		respondError(w, 400, "Upload frame image fail!")
		return
	}
	templateDefault.FrameURI = frameUrl

	yamlData, err := yaml.Marshal(&templateDefault)
	if err != nil {
		respondError(w, 400, "Error while Marshaling!")
		return
	}
	templateName := strconv.FormatInt(utils.MakeTimestamp(), 10)
	fileName := "static/fb-templates/" + templateName + ".yaml"
	err = ioutil.WriteFile(fileName, yamlData, 777)
	if err != nil {
		respondError(w, 400, "Unable to write data into the file!")
		return
	}
	respondData(w, http.StatusOK, map[string]string{
		"data": templateName + ".yaml",
	})
	return
}

func (ps *fbImageService) handleGetFBTemplates(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir("./static/fb-templates")
	if err != nil {
		log.Fatal(err)
	}
	fileArr := []string{}
	for _, f := range files {
		fileArr = append(fileArr, f.Name())
	}
	respondData(w, http.StatusOK, fileArr)
}

func (ps *fbImageService) handleFbScdnImage(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	vars := mux.Vars(r)
	params.Set("__upstream",
		ps.upstream+vars["source"],
	)

	template := params.Get("template")
	timer := prometheus.NewTimer(opsDurationProcessed.With(prometheus.Labels{"template": template}))
	defer timer.ObserveDuration()

	tmpl, ok := ps.tmpls[template]
	if !ok {
		w.WriteHeader(400)
		w.Write([]byte("template not found"))
		return
	}
	ps._process(w, params, tmpl)
}

func (ps *fbImageService) _process(w http.ResponseWriter, params url.Values, tmpl *ImageTemplate) {
	upstream := params.Get("__upstream")
	b, err := downloader.Download(upstream, "facebook")
	if err != nil {
		err2, ok := err.(*downloader.DownloadError)
		if ok && err2.Code == 404 {
			http.Error(w, "not found", 404)
		} else {
			http.Error(w, "service unavailable", err2.Code)
		}
		return
	}

	src, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	price, err := strconv.Atoi(params.Get("price"))
	if err == nil {
		promotionPrice, err := strconv.Atoi(params.Get("promotion_price"))
		if err != nil || promotionPrice <= 0 || promotionPrice > price {
			promotionPrice = price
		}
		img := tmpl.GenerateFromImage(src, price, promotionPrice)
		w.Header().Set("content-type", "image/jpeg")
		w.Write(img)
	} else {
		img := tmpl.GenerateFromImageNotPrice(src)
		w.Header().Set("content-type", "image/jpeg")
		w.Write(img)
	}
}
