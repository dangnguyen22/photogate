package appqr

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gitlab.sendo.vn/iaas-cc/api-utils/restapi/jwtauthen"
	jwtmux "gitlab.sendo.vn/iaas-cc/api-utils/restapi/jwtauthen/mux"
	"gitlab.sendo.vn/system/photogate/logger"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
	"gorm.io/gorm"
)

var (
	TemplateName = "template"
)

var opsDurationProcessed = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "photogate_qr_response_duration_time",
	Help: "Duration of QR generate image requests",
}, []string{TemplateName})

func init() {
	viper.SetDefault("qr.prefix", "http://localhost:8080/qr/")
	viper.SetDefault("qr.loglevel", "debug")
}

func init() {
	prometheus.Register(opsDurationProcessed)
}

type qrService struct {
	mr *mux.Router
	ir *mux.Router

	tmpls map[string]*template

	log zerolog.Logger
}

type qrBodyReq struct {
	Payload  string `json:"payload"`
	Template string `json:"template"`
}

func (qr *qrService) handleReady(w http.ResponseWriter, r *http.Request) {
	respondData(w, http.StatusOK, "OK")
}

func (qr *qrService) getListQR(res http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	page := 1
	pageSize := 10
	keyword := ""
	if query["page"] != nil && query["page_size"] != nil {
		page, _ = strconv.Atoi(query["page"][0])
		pageSize, _ = strconv.Atoi(query["page_size"][0])
	}
	if query["q"] != nil {
		keyword = strings.TrimSpace(query["q"][0])
	}
	limit := pageSize
	offset := page*pageSize - pageSize
	qrRecords, err := findAll(limit, offset, keyword)
	var qrRecordsReq []QrRecordRequest
	prefix := viper.GetViper().GetString("qr.prefix")

	for _, record := range qrRecords {
		qrRecord := QrRecordRequest{
			ID:       chunkEncode(record.ID),
			Payload:  record.Payload,
			Template: record.Template,
			Dtime:    record.Dtime,
			Ctime:    record.Ctime,
			Prefix:   prefix,
		}
		qrRecordsReq = append(qrRecordsReq, qrRecord)
	}
	qr.log.Debug().Msg("get list qr record")
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
	} else {
		respondData(res, http.StatusOK, qrRecordsReq)
	}
}

func respondData(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondError(w http.ResponseWriter, code int, msg string) {
	respondData(w, code, map[string]string{"error": msg})
}

func NewQrService(templateFs fs.FS) (*qrService, error) {
	initDatabase()

	mr := mux.NewRouter()
	ir := mux.NewRouter()

	log := logger.NamedLogger("qr").Level(logger.GetLogLevel("qr.loglevel"))

	tmpls, err := loadTemplates(log, templateFs, "qr-templates")
	if err != nil {
		return nil, err
	}

	s := &qrService{
		mr:    mr,
		ir:    ir,
		tmpls: tmpls,
	}

	mr.Methods("GET").Path("/{code}/{size}").HandlerFunc(s.handleQrGenImage)
	mr.Methods("GET").Path("/{code}").HandlerFunc(s.handleQrGenImage)

	ir.Methods("GET").Path("/ready").HandlerFunc(s.handleReady).Name("READY")
	ir.Methods("POST").Path("/generate").HandlerFunc(s.handleQrGenLink).Name("GENERATE_QR")
	ir.Methods("GET").Path("/").HandlerFunc(s.getListQR).Name("GET_QRS")
	ir.Methods("GET").Path("/{qr_id}").HandlerFunc(s.getQrById).Name("GET_QR")
	ir.Methods("POST").Path("/create").HandlerFunc(s.handleCreateQr).Name("CREATE_QR")
	ir.Methods("PUT").Path("/update/{qr_id}").HandlerFunc(s.handleUpdateQr).Name("UPDATE_QR")
	ir.Methods("DELETE").Path("/{qr_id}").HandlerFunc(s.removeQrById).Name("DELETE_QR")
	ir.Use(
		jwtmux.NewJwtAuthenticationMiddleware(
			jwtmux.AllowByName("", "READY"),
			jwtmux.AllowByName("", "GET_QRS"),
			jwtmux.AllowByName("", "GENERATE_QR"),
			jwtmux.AllowByName("", "GET_QR"),
			jwtmux.AllowByName("", "CREATE_QR"),
			jwtmux.AllowByFunc(checkAllowedRole),
			jwtmux.WithCustomClaims(&jwtauthen.XClaims{}),
		),
	)

	return s, nil
}

func checkAllowedRole(r *http.Request, c jwtauthen.Claims) bool {
	route := mux.CurrentRoute(r)
	switch name := route.GetName(); name {
	case "GET_QRS", "GET_QR":
		requireRole := "photogate.qr.viewer"
		requireAdminRole := "photogate.qr.admin"
		return c.ContainRole(requireRole) || c.ContainRole(requireAdminRole)
	case "GENERATE_QR", "CREATE_QR", "UPDATE_QR", "DELETE_QR":
		requireAdminRole := "photogate.qr.admin"
		return c.ContainRole(requireAdminRole)
	}
	return false
}

func (qr *qrService) MainHandler() http.Handler {
	return qr.mr
}

func (qr *qrService) InternalHandler() http.Handler {
	return qr.ir
}

func createQRLink(payload, template string) (string, error) {
	id, err := addNewShortHand(payload, template)
	if err != nil {
		return "", err
	}

	prefix := viper.GetViper().GetString("qr.prefix")
	return prefix + chunkEncode(id), nil
}

func (qr *qrService) handleQrGenLink(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	payload := r.Form.Get("payload")
	if payload == "" {
		http.Error(w, `{"error":"no payload"}`, 400)
		return
	}
	template := r.Form.Get("template")
	qr.log.Debug().Str("template", template).Str("payload", payload).Msg("generate qr link")

	s, err := createQRLink(payload, template)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte(s))
}

func pushPrefixUrl(qrRecord QrRecordRequest) QrRecordRequest {
	prefix := viper.GetViper().GetString("qr.prefix")
	qrRecord.Prefix = prefix
	return qrRecord
}

func (qr *qrService) getQrById(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	qrId := vars["qr_id"]
	id, err := chunkDecode(qrId)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}
	qr.log.Debug().Str("id", strconv.FormatUint(id, 10)).Msg("get qr record by id")
	qrRecord, err := findByID(id)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}
	respondData(res, http.StatusOK, pushPrefixUrl(parseIdToQrID(qrRecord)))
}

func (qr *qrService) removeQrById(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	qrId := vars["qr_id"]
	id, err := chunkDecode(qrId)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}
	qr.log.Debug().Str("id", strconv.FormatUint(id, 10)).Msg("remove qr record by id")
	err = removeQrRecordById(id)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}
	respondData(res, http.StatusOK, "success")
}

func (qr *qrService) handleCreateQr(res http.ResponseWriter, req *http.Request) {
	var qrBody qrBodyReq
	err := json.NewDecoder(req.Body).Decode(&qrBody)
	if err != nil || qrBody.Payload == "" {
		http.Error(res, `{"error":"no payload"}`, 400)
		return
	}
	qr.log.Debug().Str("template", qrBody.Template).Str("payload", qrBody.Payload).Msg("create qr record")

	id, err := addNewShortHand(qrBody.Payload, qrBody.Template)
	if err != nil {
		return
	}

	qrRecord, err := findByID(id)
	if err != nil {
		return
	}
	respondData(res, http.StatusOK, parseIdToQrID(qrRecord))
}

func (qr *qrService) handleUpdateQr(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	qrId := vars["qr_id"]
	var qrBody qrBodyReq
	err := json.NewDecoder(req.Body).Decode(&qrBody)
	if err != nil || qrBody.Payload == "" {
		http.Error(res, `{"error":"no payload"}`, 400)
		return
	}
	id, err := chunkDecode(qrId)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}

	qr.log.Debug().Str("id", strconv.FormatUint(id, 10)).Str("template", qrBody.Template).Str("payload", qrBody.Payload).Msg("update qr record")

	err = updateQrRecordById(id, qrBody.Payload, qrBody.Template)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}

	qrRecord, err := findByID(id)
	if err != nil {
		respondError(res, http.StatusBadGateway, err.Error())
		return
	}
	respondData(res, http.StatusOK, parseIdToQrID(qrRecord))
}

func (qr *qrService) _getTemplateOrDefault(t string) *template {
	tm, ok := qr.tmpls[t]
	if !ok {
		tm = qr.tmpls["default"]
	}
	return tm
}

// generate QR by a template
func (qr *qrService) generateQr(sh QrRecord, size int) ([]byte, error) {
	tm := qr._getTemplateOrDefault(sh.Template)

	img, err := tm.Render(sh.Payload, size)
	if err != nil {
		return nil, err
	}
	return imghelper.Img2pngBuf(img), nil
}

func (qr *qrService) handleQrGenImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	size, _ := strconv.Atoi(vars["size"])
	timerEmptyTemplate := prometheus.NewTimer(opsDurationProcessed.With(prometheus.Labels{"template": "None"}))

	var sh QrRecord
	{
		defer timerEmptyTemplate.ObserveDuration()
		n, err := chunkDecode(code)
		if err != nil {
			w.WriteHeader(400)
			w.Write(imghelper.Empty1x1_PNG)
			return
		}

		sh, err = getShortHandById(n)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				w.WriteHeader(404)
			} else {
				w.WriteHeader(500)
				qr.log.Error().Err(err).Msg("get qr payload")
			}
			w.Write(imghelper.Empty1x1_PNG)
			return
		}
	}
	timer := prometheus.NewTimer(opsDurationProcessed.With(prometheus.Labels{"template": sh.Template}))
	start := time.Now()
	defer func() {
		qr.log.Debug().
			Str("template", sh.Template).
			Str("payload", sh.Payload).
			Dur("duration", time.Since(start)).
			Msg("generate qr image")
		timer.ObserveDuration()
	}()
	if b, err := qr.generateQr(sh, size); err != nil {
		w.WriteHeader(500)
		w.Write(imghelper.Empty1x1_PNG)
	} else {
		w.Header().Add("content-type", "image/png")
		w.Write(b)
	}
}
