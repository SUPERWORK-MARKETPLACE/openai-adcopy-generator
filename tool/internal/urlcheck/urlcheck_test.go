package urlcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckClassifiesResults(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer okSrv.Close()
	notFoundSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer notFoundSrv.Close()
	redirSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, okSrv.URL+"/landed", http.StatusFound)
	}))
	defer redirSrv.Close()

	res := Check([]string{okSrv.URL, notFoundSrv.URL, redirSrv.URL, "http://127.0.0.1:1/dead"}, 3*time.Second)

	if !res[0].OK || res[0].Status != 200 {
		t.Fatalf("ok url misclassified: %+v", res[0])
	}
	if res[0].HTTPS {
		t.Errorf("httptest server is http — HTTPS must be false")
	}
	if res[1].OK || res[1].Status != 404 {
		t.Fatalf("404 must not be OK: %+v", res[1])
	}
	if !res[2].OK {
		t.Fatalf("redirected url must be OK: %+v", res[2])
	}
	if res[2].RedirectedDomain {
		// both servers run on 127.0.0.1 — hostname unchanged, only the port differs
		t.Errorf("same-host redirect must not flag RedirectedDomain: %+v", res[2])
	}
	if res[2].FinalURL == redirSrv.URL {
		t.Errorf("redirect final URL not captured: %+v", res[2])
	}
	if res[3].OK || res[3].Error == "" {
		t.Fatalf("dead server must report error: %+v", res[3])
	}
}
