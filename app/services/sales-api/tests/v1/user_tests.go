package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"testing"

	"github.com/tcmhoang/sservices/app/services/sales-api/handlers"
	"github.com/tcmhoang/sservices/business/data/tests"
)

type UserTests struct {
	app        http.Handler
	userToken  string
	adminToken string
}

func Test_Users(t *testing.T) {
	t.Parallel()

	test := tests.NewTest(t, c)
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
			t.Error(string(debug.Stack()))
		}
		test.Teardown()
	}()

	shutdown := make(chan os.Signal, 1)
	tests := UserTests{
		app: handlers.APIMux(handlers.APIMuxConfig{
			Shutdown: shutdown,
			Log:      test.Log,
			Auth:     test.Auth,
			DB:       test.DB,
		}),
		userToken:  test.Token("user@example.com", "gophers"),
		adminToken: test.Token("admin@example.com", "gophers"),
	}

	t.Run("getToken404", tests.getToken404())
	t.Run("getToken200", tests.getToken200())

}

func (ut *UserTests) getToken404() func(t *testing.T) {
	return func(t *testing.T) {
		url := "/v1/users/token/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.SetBasicAuth("unknown@example.com", "some-password")
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Should receive a status code of 404 for the response : %d", w.Code)
		}
	}
}

func (ut *UserTests) getToken200() func(t *testing.T) {
	return func(t *testing.T) {
		url := "/v1/users/token/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.SetBasicAuth("admin@example.com", "gophers")
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Should receive a status code of 200 for the response : %d", w.Code)
		}

		var got struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("Should be able to unmarshal the response : %s", err)
		}
	}
}
