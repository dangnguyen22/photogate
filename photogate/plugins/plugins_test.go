package plugins

import (
	"image/color"
	"testing"

	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/require"
	"gitlab.sendo.vn/system/photogate/downloader"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
)

func init() {
	downloader.Init()
}

func TestPlugins(t *testing.T) {
	cfg := []map[string]interface{}{
		{
			"type":  "image",
			"image": "https://media3.scdn.vn/img4/2022/03_10/dS5ULm9X1jHE7sCeG17J.png",
		},
	}

	plugins, err := NewPluginsFromConfig(cfg)
	require.NoError(t, err)

	_ = plugins
	err = plugins.Configure()
	require.NoError(t, err)
}

func TestPluginsRender(t *testing.T) {
	plugins := Plugins([]Plugin{
		&QrPlugin{
			Text: "https://sendo.vn/sendofarm",
			// Color: "000",
			Anchor:   FPoint{0.5, 0.4},
			Size:     0.8,
			Recovery: qrcode.Highest,
		},
		&ImagePlugin{
			Image: "https://media3.scdn.vn/img4/2022/03_10/dS5ULm9X1jHE7sCeG17J.png",
			Rect:  FRectangle{0, 0, 1, 1},
		},
	})

	err := plugins.Configure()
	require.NoError(t, err)

	dc := imghelper.InitDrawingContext(400, 500, color.White)
	err = plugins.Execute(dc)
	require.NoError(t, err)

	_ = dc
	// dc.SavePNG("test.png")
}
