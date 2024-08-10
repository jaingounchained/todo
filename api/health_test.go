package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHealthAPI(t *testing.T) {
	// start test server and send request
	server := NewGinHandler(nil, nil, nil)
	recorder := httptest.NewRecorder()

	url := "/health"
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	server.router.ServeHTTP(recorder, request)

	data, err := io.ReadAll(recorder.Body)
	require.NoError(t, err)

	require.Equal(t, data, []byte("\"OK\""))
}
