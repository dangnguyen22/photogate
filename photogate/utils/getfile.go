package utils

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gitlab.sendo.vn/system/photogate/downloader"
)

var (
	staticFs   fs.FS
	staticRoot string

	defaultClient = http.Client{
		Timeout: time.Second * 10,
	}

	GetNotFound   = errors.New("file not found")
	UpstreamError = errors.New("upstream error")
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	staticRoot = path.Join(path.Dir(filename), "../static")
}

func Init(fs fs.FS) {
	staticFs = fs
}

func SimpleGetFile(uri string) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return downloader.Download(uri, "misc")
	} else if u.Scheme == "local" {
		p := path.Join(staticRoot, u.Path)
		return os.ReadFile(p)
	} else if staticFs != nil {
		return fs.ReadFile(staticFs, strings.TrimPrefix(uri, "/"))
	} else {
		return nil, fmt.Errorf(`unknown how to get "%s"`, uri)
	}
}
