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
	"time"

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

func TestUsers(t *testing.T) {
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
	t.Run("crudUsers", tests.crudUser())

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
				Name:    "Conrad",
				Address: "tcmhoang@outlook.com",
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

func (ut *UserTests) crudUser() func(t *testing.T) {
	return func(t *testing.T) {
		usr := ut.postUser201(t)
		defer ut.deleteUser204(t, usr.ID.String())

		ut.postUser409(t, usr)

		ut.getUser200(t, usr.ID.String())
		ut.putUser200(t, usr.ID.String())
		ut.putUser401(t, usr.ID.String())
	}
}

func (ut *UserTests) postUser201(t *testing.T) user.User {
	nu := user.NewUser{
		Name: "Conrad Hoang",
		Email: mail.Address{
			Name:    "Conrad Hoang",
			Address: "tcmhoang@outlook.com",
		},
		Roles:           []string{"ADMIN"},
		Password:        "gophers",
		PasswordConfirm: "gophers",
	}

	body, err := json.Marshal(&nu)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("Should receive a status code of 201 for the response : %d", w.Code)
	}

	var newUsr user.User
	if err := json.NewDecoder(w.Body).Decode(&newUsr); err != nil {
		t.Fatalf("Should be able to unmarshal the response : %s", err)
	}

	email, err := mail.ParseAddress("tcmhoang@outlook.com")
	if err != nil {
		t.Fatalf("Should be able to parse email : %s", err)
	}

	exp := newUsr
	exp.Name = "Conrad Hoang"
	exp.Email = *email
	exp.Roles = []string{"ADMIN"}

	if diff := cmp.Diff(newUsr, exp); diff != "" {
		t.Fatalf("Should get the expected result, diff:\n%s", diff)
	}

	return newUsr
}

func (ut *UserTests) deleteUser204(t *testing.T, id string) {
	url := fmt.Sprintf("/v1/users/%s", id)

	r := httptest.NewRequest(http.MethodDelete, url, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Should receive a status code of 204 for the response : %d", w.Code)
	}
}

func (ut *UserTests) postUser409(t *testing.T, usr user.User) {
	nu := user.NewUser{
		Name:            usr.Name,
		Email:           usr.Email,
		Roles:           usr.Roles,
		Password:        "gophers",
		PasswordConfirm: "gophers",
	}

	body, err := json.Marshal(&nu)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("Should receive a status code of 409 for the response : %d", w.Code)
	}
}

func (ut *UserTests) getUser200(t *testing.T, id string) {
	url := fmt.Sprintf("/v1/users/%s", id)

	r := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Should receive a status code of 200 for the response : %d", w.Code)
	}

	var got struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		Email        string    `json:"email"`
		Roles        []string  `json:"roles"`
		PasswordHash []byte    `json:"-"`
		Department   string    `json:"department"`
		Enabled      bool      `json:"enabled"`
		DateCreated  time.Time `json:"dateCreated"`
		DateUpdated  time.Time `json:"dateUpdated"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("Should be able to unmarshal the response : %s", err)
	}

	email, err := mail.ParseAddress("tcmhoang@outlook.com")
	if err != nil {
		t.Fatalf("Should be able to parse email : %s", err)
	}

	exp := got
	exp.ID = id
	exp.Name = "Conrad Hoang"
	exp.Email = email.Address
	exp.Roles = []string{"ADMIN"}

	if diff := cmp.Diff(got, exp); diff != "" {
		t.Errorf("Should get the expected result, Diff:\n%s", diff)
	}
}

func (ut *UserTests) putUser200(t *testing.T, id string) {
	u := user.UpdateUser{
		Name: tests.StringPointer("Conrad Hoang"),
	}
	body, err := json.Marshal(&u)
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("/v1/users/%s", id)

	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Should receive a status code of 200 for the response : %d", w.Code)
	}

	r = httptest.NewRequest(http.MethodGet, "/v1/users/"+id, nil)
	w = httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Should receive a status code of 200 for the retrieve : %d", w.Code)
	}

	var ru user.User
	if err := json.NewDecoder(w.Body).Decode(&ru); err != nil {
		t.Fatalf("Should be able to unmarshal the response : %s", err)
	}

	if ru.Name != "Conrad Hoang" {
		t.Fatalf("Should see an updated Name : got %q want %q", ru.Name, "Conrad Hoang")
	}

	email, err := mail.ParseAddress("tcmhoang@outlook.com")
	if err != nil {
		t.Fatalf("Should be able to parse email : %s", err)
	}

	if ru.Email.String() != email.Address {
		t.Fatalf("Should not affect other fields like Email : got %q want %q", ru.Email, "tcmhoang@outlook.com")
	}
}

func (ut *UserTests) putUser401(t *testing.T, id string) {
	u := user.NewUser{
		Name: "Bill Ken",
	}
	body, err := json.Marshal(&u)
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("/v1/users/%s", id)

	r := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.userToken)
	ut.app.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Should receive a status code of 401 for the response : %d", w.Code)
	}
}
