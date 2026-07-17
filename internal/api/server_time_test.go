package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestServerTimeHeader(t *testing.T) {
	before := time.Now().UnixMilli()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	(&API{}).serverTime(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, request)
	after := time.Now().UnixMilli()

	value, err := strconv.ParseInt(recorder.Header().Get("X-CFScan-Server-Time"), 10, 64)
	if err != nil {
		t.Fatalf("parse server time header: %v", err)
	}
	if value < before || value > after {
		t.Fatalf("server time %d outside request bounds %d..%d", value, before, after)
	}
}
