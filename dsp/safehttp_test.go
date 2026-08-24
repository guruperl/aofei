package dsp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

const safeTestOrigin = "http://8.8.8.8"

type safeTestRoundTripper struct {
	handler http.Handler
}

func (t safeTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	result := make(chan *http.Response, 1)
	go func() {
		recorder := httptest.NewRecorder()
		t.handler.ServeHTTP(recorder, req)
		response := recorder.Result()
		response.Request = req
		result <- response
	}()
	select {
	case response := <-result:
		return response, nil
	case <-req.Context().Done():
		return nil, fmt.Errorf("safe test transport: %w", req.Context().Err())
	}
}

func (safeTestRoundTripper) SafeHTTPNonNetworkTransport() {}

func safeTestClient(server *httptest.Server) *http.Client {
	return &http.Client{Transport: safeTestRoundTripper{handler: server.Config.Handler}}
}
