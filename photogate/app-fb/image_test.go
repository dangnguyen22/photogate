package appfb

import (
	"bytes"
	"image"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.sendo.vn/system/photogate/downloader"
	"gitlab.sendo.vn/system/photogate/plugins/imghelper"
)

const (
	testImageJpg  = "tests/test-500x500.jpg"
	testImagePng  = "tests/test-500x500.png"
	testImageWebp = "tests/test-500x500.webp"
	testImageHeic = "tests/test-500x500.heic"
	testImageGif  = "tests/test-500x500.gif"
)

var (
	defaultTestConfig = `
frameURI: local:/2020-02/frame.png
priceOnly:
  top: 0.91
  right: 0.075
  fontSize: 12
  fontDpi: 120
  fontUri: local:/2020-02/UTMAVO-BOLD.TTF
priceOrig:
  top: 0.885
  right: 0.072
  fontSize: 10.5
  fontDpi: 120
  fontUri: local:/2020-02/UTMAVO-REGULAR.TTF
  strikeThrough: 0.1
pricePromo:
  top: 0.92
  right: 0.075
  fontSize: 14
  fontDpi: 120
`
)

func init() {
	downloader.Init()
}

func loadTestTemplate(t testing.TB, s string) *ImageTemplate {
	itc, err := loadImageTemplateConfig(s)
	require.NoError(t, err)
	it, err := NewImageTemplate(itc)
	require.NoError(t, err)
	return it
}

func TestImageTemplate_Generate(t *testing.T) {
	it := loadTestTemplate(t, defaultTestConfig)
	buf, err := ioutil.ReadFile(testImageJpg)
	require.NoError(t, err)

	t.Run("only", func(t *testing.T) {
		_, err = it.Generate(buf, 1234567, 1234567)
		require.NoError(t, err)
	})
	t.Run("promo", func(t *testing.T) {
		_, err = it.Generate(buf, 1234567, 123456)
		require.NoError(t, err)
	})
}

func TestImageFormats(t *testing.T) {
	it := loadTestTemplate(t, defaultTestConfig)

	tests := []string{
		testImageJpg,
		testImagePng,
		testImageGif,
		testImageWebp,
		testImageHeic,
	}

	for _, s := range tests {
		t.Run(path.Ext(s)[1:], func(t *testing.T) {
			buf, err := ioutil.ReadFile(s)
			require.NoError(t, err)
			_, err = it.Generate(buf, 1234567, 123456)
			require.NoError(t, err)
		})
	}
}

func BenchmarkTemplate_Generate(b *testing.B) {
	it := loadTestTemplate(b, defaultTestConfig)
	buf, err := ioutil.ReadFile(testImageJpg)
	require.NoError(b, err)

	img, _, _ := image.Decode(bytes.NewReader(buf))
	jpgBuff := imghelper.Img2jpegBuf(img)
	pngBuff := imghelper.Img2pngBuf(img)

	params := []struct {
		Name   string
		Buf    []byte
		P1, P2 int
	}{
		{"jpg-priceonly", jpgBuff, 123450, 123450},
		{"jpg-pricepromo", jpgBuff, 123450, 103450},
		{"png-priceonly", pngBuff, 123450, 123450},
		{"png-pricepromo", pngBuff, 123450, 103450},
	}

	b.Run("single", func(b *testing.B) {
		for _, c := range params {
			b.Run(c.Name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					out, err := it.Generate(c.Buf, c.P1, c.P2)
					require.NoError(b, err)
					require.Greater(b, len(out), 10000)
				}
			})
		}
	})

	b.Run("parallel", func(b *testing.B) {
		for _, c := range params {
			b.Run(c.Name, func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						out, err := it.Generate(c.Buf, c.P1, c.P2)
						require.NoError(b, err)
						require.Greater(b, len(out), 10000)
					}
				})
			})
		}
	})
}
