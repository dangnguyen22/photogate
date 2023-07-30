package downloader

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloader(t *testing.T) {
	dl := newDownloadService(2)
	_, err := dl.Download("https://google.com", "")
	require.NoError(t, err)

	_, err = dl.Download("https://google.com/notfound", "")
	require.Error(t, err)

	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			dl.Download("https://google.com", "")
			wg.Done()
		}()
	}
	wg.Wait()
}
