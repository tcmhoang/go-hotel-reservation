package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/tcmhoang/sservices/app/services/sales-api/handlers"
	"github.com/tcmhoang/sservices/business/data/store/user"
	"github.com/tcmhoang/sservices/business/data/tests"
	"github.com/tcmhoang/sservices/business/sys/validation"
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

	seed := func(ctx context.Context, usrCore *user.Store) ([]user.User, error) {
		usrs, err := usrCore.Query(ctx, 1, 2)
		if err != nil {
			return nil, fmt.Errorf("seeding users : %w", err)
		}

		return usrs, nil
	}

	t.Log("Go seeding ...")

	usrs, err := seed(context.Background(), user.NewStore(test.Log, test.DB))
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	t.Run("getToken404", tests.getToken404())
	t.Run("getToken200", tests.getToken200())
	t.Run("postUser400", tests.postUser400())
	t.Run("postUser401", tests.postUser401())
	t.Run("postNoAuth401", tests.postNoAuth401())
	t.Run("getUser400", tests.getUser400())
	t.Run("getUser401", tests.getUser401(usrs))
	t.Run("getUser404", tests.getUser404())
	t.Run("deleteUserNotFound", tests.deleteUserNotFound())
	t.Run("putUser404", tests.putUser404())
	t.Run("getUsers200", tests.getUsers200(usrs))

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

func (ut *UserTests) postUser400() func(t *testing.T) {
	return func(t *testing.T) {
		usr := user.NewUser{
			Email: mail.Address{
				Name:    "Bill",
				Address: "bill@ardanlabs.com",
			},
		}

		body, err := json.Marshal(usr)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Should receive a status code of 400 for the response : %d", w.Code)
		}

		var got validation.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("Should be able to unmarshal the response to an error type : %s", err)
		}

		fields := validation.FieldErrors{
			{Field: "name", Error: "name is a required field"},
			{Field: "roles", Error: "roles is a required field"},
			{Field: "password", Error: "password is a required field"},
		}
		exp := validation.ErrorResponse{
			Error:  "data validation error",
			Fields: fields.Fields(),
		}
		sorter := cmpopts.SortSlices(func(a, b validation.FieldError) bool {
			return a.Field < b.Field
		})

		if diff := cmp.Diff(got, exp, sorter); diff != "" {
			t.Fatalf("Should get the expected result, diff:\n%s", diff)
		}
	}
}

func (ut *UserTests) postUser401() func(t *testing.T) {
	return func(t *testing.T) {
		body, err := json.Marshal(&user.NewUser{})
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.userToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Should receive a status code of 401 for the response : %d", w.Code)
		}
	}
}

func (ut *UserTests) postNoAuth401() func(t *testing.T) {
	return func(t *testing.T) {
		body, err := json.Marshal(&user.NewUser{})
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Should receive a status code of 401 for the response : %d", w.Code)
		}
	}
}

func (ut *UserTests) getUser400() func(t *testing.T) {
	return func(t *testing.T) {
		url := fmt.Sprintf("/v1/users/%d", 12345)

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Should receive a status code of 400 for the response : %d", w.Code)
		}

		got := w.Body.String()
		exp := `{"error":"ID is not in its proper form"}`
		if got != exp {
			t.Logf("got: %v", got)
			t.Logf("exp: %v", exp)
			t.Errorf("Should get the expected result")
		}
	}
}

func (ut *UserTests) getUser401(usrs []user.User) func(t *testing.T) {
	return func(t *testing.T) {
		url := fmt.Sprintf("/v1/users/%s", usrs[0].ID)

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.userToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Should receive a status code of 401 for the response : %d", w.Code)
		}

		recv := w.Body.String()
		resp := `{"error":"Unauthorized"}`
		if resp != recv {
			t.Log("got:", recv)
			t.Log("exp:", resp)
			t.Fatalf("Should get the expected result.")
		}
		url = fmt.Sprintf("/v1/users/%s", usrs[1].ID)

		r = httptest.NewRequest(http.MethodGet, url, nil)
		w = httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.userToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Should receive a status code of 200 for the response : %d", w.Code)
		}
	}
}

func (ut *UserTests) getUser404() func(t *testing.T) {
	return func(t *testing.T) {
		url := fmt.Sprintf("/v1/users/%s", "c50a5d66-3c4d-453f-af3f-bc960ed1a503")

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Should receive a status code of 404 for the response : %d", w.Code)
		}

		got := w.Body.String()
		exp := "not found"
		if !strings.Contains(got, exp) {
			t.Logf("got: %v", got)
			t.Logf("exp: %v", exp)
			t.Errorf("Should get the expected result")
		}
	}
}

func (ut *UserTests) deleteUserNotFound() func(t *testing.T) {
	return func(t *testing.T) {
		url := fmt.Sprintf("/v1/users/%s", "a71f77b2-b1ae-4964-a847-f9eecba09d74")

		r := httptest.NewRequest(http.MethodDelete, url, nil)
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Fatalf("Should receive a status code of 204 for the response : %d", w.Code)
		}
	}
}

func (ut *UserTests) putUser404() func(t *testing.T) {
	return func(t *testing.T) {
		url := fmt.Sprintf("/v1/users/%s", "3097c45e-780a-421b-9eae-43c2fda2bf14")

		u := user.UpdateUser{
			Name: tests.StringPointer("Doesn't Exist"),
		}
		body, err := json.Marshal(&u)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Should receive a status code of 404 for the response : %d", w.Code)
		}

		got := w.Body.String()
		exp := "not found"
		if !strings.Contains(got, exp) {
			t.Logf("got: %v", got)
			t.Logf("exp: %v", exp)
			t.Errorf("Should get the expected result")
		}
	}
}

func (ut *UserTests) getUsers200(usrs []user.User) func(t *testing.T) {
	return func(t *testing.T) {
		url := "/v1/users?page=1&rows=2"

		r := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		r.Header.Set("Authorization", "Bearer "+ut.adminToken)
		ut.app.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Should receive a status code of 200 for the response : %d", w.Code)
		}

		var users []user.User
		if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
			t.Fatalf("Should be able to unmarshal the response : %s", err)
		}

		if len(users) != len(usrs) {
			t.Log("got:", len(users))
			t.Log("exp:", len(usrs))
			t.Error("Should get the right total")
		}
	}
}
