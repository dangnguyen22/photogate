package appfb

import (
	"encoding/json"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

func (ps *fbImageService) handleDebugListTemplates(w http.ResponseWriter, r *http.Request) {
	tmpls := []string{}
	for t := range ps.tmpls {
		tmpls = append(tmpls, t)
	}

	b, err := json.Marshal(tmpls)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(b)
}

func (ps *fbImageService) handleDebugConfig(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	tmpl, ok := ps.tmpls[template]
	if !ok {
		w.WriteHeader(400)
		w.Write([]byte("template not found"))
		return
	}

	enc := yaml.NewEncoder(w)
	if err := enc.Encode(tmpl.cfg); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
}

func (ps *fbImageService) handleDebugImage(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	params.Set("__upstream", params.Get("url"))

	cfg, err := loadImageTemplateConfig(strings.TrimSpace(params.Get("config")))
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	tmpl, err := NewImageTemplate(cfg)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	ps._process(w, params, tmpl)
}
