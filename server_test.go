package kocha

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestServer(t *testing.T) {
	oldAppConfig := appConfig
	appConfig = newTestAppConfig()
	defer func() {
		appConfig = oldAppConfig
	}()
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status := w.Code
	if !reflect.DeepEqual(status, http.StatusOK) {
		t.Errorf("Expect %v, but %v", http.StatusOK, status)
	}
	body := w.Body.String()
	expected := "tmpl1"
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Expect %v, but %v", expected, body)
	}

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/user/7", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status = w.Code
	if !reflect.DeepEqual(status, http.StatusOK) {
		t.Errorf("Expect %v, but %v", http.StatusOK, status)
	}
	body = w.Body.String()
	expected = "tmpl2-7"
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Expect %v, but %v", expected, body)
	}

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/2013/07/19/user/naoina", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status = w.Code
	if !reflect.DeepEqual(status, http.StatusOK) {
		t.Errorf("Expect %v, but %v", http.StatusOK, status)
	}
	body = w.Body.String()
	expected = "tmpl3-naoina-2013-7-19"
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Expect %v, but %v", expected, body)
	}

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status = w.Code
	if !reflect.DeepEqual(status, http.StatusNotFound) {
		t.Errorf("Expect %v, but %v", http.StatusOK, status)
	}

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status = w.Code
	if !reflect.DeepEqual(status, http.StatusNotFound) {
		t.Errorf("Expect %v, but %v", http.StatusOK, status)
	}

	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/error", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler(w, req)
	status = w.Code
	if !reflect.DeepEqual(status, http.StatusInternalServerError) {
		t.Errorf("Expect %v, but %v", http.StatusInternalServerError, status)
	}
}