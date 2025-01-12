package main

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckAPI(t *testing.T) {
	app := createTestApp()
	recorder := httptest.NewRecorder()

	url := "/v1/healthcheck"
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	app.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	// no need to check other things as of now.
}
