package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithEmbeddedWebServer(t *testing.T) {

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/500":
				http.Error(w, "missing key", http.StatusInternalServerError)
			case "/404":
				http.Error(w, "missing key", http.StatusNotFound)
			default:
				_, _ = w.Write([]byte("Hello"))
			}
		}))

	url := ts.URL

	var webClient *WebClient
	var measure *HTTPMeasure

	webClient, _ = NewWebClient(&Config{Target: fmt.Sprintf("%s/500", url)})
	measure = webClient.DoMeasure()

	if !measure.IsFailure || measure.StatusCode != 500 {
		t.Errorf("Request to server should have failed, 500")
	}

	webClient, _ = NewWebClient(&Config{Target: fmt.Sprintf("%s/200", url)})
	measure = webClient.DoMeasure()

	if measure.IsFailure || measure.StatusCode != 200 {
		t.Errorf("Request to server should have succeed, 200")
	}

	ts.Close()

	measure = webClient.DoMeasure()

	if !measure.IsFailure {
		t.Errorf("Request to server should have failed")
	}

}
