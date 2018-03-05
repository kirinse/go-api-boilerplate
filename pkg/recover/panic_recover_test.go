/*
Package recover allows to recover from panic
*/
package recover

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	t.Parallel()

	recover := New()

	if recover == nil {
		t.Fail()
	}
}

func TestRecoverHandler(t *testing.T) {
	t.Parallel()

	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	recover := New()
	handler := recover.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler dod not recoverd")
	}
}

func TestRecoverHandlerWithLogger(t *testing.T) {
	t.Parallel()

	paniced := false
	defer func() {
		if rcv := recover(); rcv != nil {
			paniced = true
		}
	}()

	recover := WithLogger(New(), golog.New("debug"))
	handler := recover.RecoverHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("error")
	}))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(w, req)

	if paniced == true {
		t.Error("RecoverHandler dod not recoverd")
	}
}