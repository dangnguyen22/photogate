package appqr

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gitlab.sendo.vn/system/photogate/downloader"
)

func init() {
	downloader.Init()
}

func TestQrConfig(t *testing.T) {
	c, err := loadTemplate("hello", []byte(`
allWidths: [128, 256, 512]
plugins:
- type: image
  image: https://media3.scdn.vn/img4/2022/03_10/dS5ULm9X1jHE7sCeG17J.png
`))

	require.NoError(t, err)
	require.EqualValues(t, []int{128, 256, 512}, c.AllWidths)
	require.Len(t, c._plugins, 1)
}

func TestQrTemplate(t *testing.T) {
	static := fstest.MapFS{
		"hello": &fstest.MapFile{
			Data: []byte("abcdef"),
		},
		"tmpls/x/something.yaml": &fstest.MapFile{
			Data: []byte(`
widthHeightRatio: 0.8
allWidths: [128, 256, 512]
plugins:
- type: image
  image: https://media3.scdn.vn/img4/2022/03_10/dS5ULm9X1jHE7sCeG17J.png
`),
		},
	}
	err := fstest.TestFS(static, "hello")
	require.NoError(t, err)

	tDir, err := fs.Sub(static, "tmpls/x")
	require.NoError(t, err)

	err = fstest.TestFS(tDir, "hello")
	require.Error(t, err)

	tmpls, err := loadTemplates(log.Logger, tDir, ".")
	require.NoError(t, err)
	require.Len(t, tmpls, 1)
	require.NotNil(t, tmpls["something"])
}
