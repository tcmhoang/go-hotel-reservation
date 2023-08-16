package user_test

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"runtime/debug"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/tcmhoang/sservices/business/data/store/user"
	"github.com/tcmhoang/sservices/business/data/tests"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/foundation/docker"
)

var c *docker.Container

func TestMain(m *testing.M) {
	var err error
	c, err = tests.InitDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tests.StopDB(c)

	m.Run()
}

func TestUser(t *testing.T) {
	t.Run("crud", crud)
	t.Run("paging", paging)

}

func crud(t *testing.T) {
	stest := tests.NewTest(t, c)
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
			t.Error(string(debug.Stack()))
		}
		stest.Teardown()
	}()

	store := user.NewStore(stest.Log, stest.DB)

	t.Log("Given the need to work with User records.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen handling a single User.", testID)
		{
			ctx := context.Background()

			email, err := mail.ParseAddress("tcmhoang@foobar.com")
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to parse email: %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to parse email.", tests.Success, testID)

			nu := user.NewUser{
				Name:            "Conrad Hoang",
				Email:           *email,
				Roles:           []string{"Admin"},
				Password:        "gophers",
				PasswordConfirm: "gophers",
			}

			usr, err := store.Create(ctx, nu)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to create user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to create user.", tests.Success, testID)

			saved, err := store.QueryByID(ctx, usr.ID)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user by ID: %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve user by ID.", tests.Success, testID)

			if usr.DateCreated.UnixMilli() != saved.DateCreated.UnixMilli() {
				t.Logf("\t\tTest %d:\tGot: %v", testID, saved.DateCreated)
				t.Logf("\t\tTest %d:\tExp: %v", testID, usr.DateCreated)
				t.Logf("\t\tTest %d:\tDiff: %v", testID, saved.DateCreated.Sub(usr.DateCreated))
				t.Fatalf("\t%s\tTest %d:\tShould get back the same date created.", tests.Failed, testID)
			}
			t.Logf("\t%s\tTest %d:\tShould get back the same date created.", tests.Success, testID)

			if usr.DateUpdated.UnixMilli() != saved.DateUpdated.UnixMilli() {
				t.Logf("\t\tTest %d:\tGot: %v", testID, saved.DateUpdated)
				t.Logf("\t\tTest %d:\tExp: %v", testID, usr.DateUpdated)
				t.Logf("\t\tTest %d:\tDiff: %v", testID, saved.DateUpdated.Sub(usr.DateUpdated))
				t.Fatalf("\t%s\tTest %d:\tShould get back the same date updated.", tests.Failed, testID)
			}
			t.Logf("\t%s\tTest %d:\tShould get back the same date updated.", tests.Success, testID)

			usr.DateCreated = time.Time{}
			usr.DateUpdated = time.Time{}
			saved.DateCreated = time.Time{}
			saved.DateUpdated = time.Time{}

			if diff := cmp.Diff(usr, saved); diff != "" {
				t.Fatalf("\t%s\tTest %d:\tShould get back the same user. Diff:\n%s", tests.Failed, testID, diff)
			}
			t.Logf("\t%s\tTest %d:\tShould get back the same user.", tests.Success, testID)

			email, err = mail.ParseAddress("tcmhoang@brewtea.com")
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to parse email: %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to parse email.", tests.Success, testID)

			upd := user.UpdateUser{
				Name:       tests.StringPointer("Foo Bar"),
				Email:      email,
				Department: tests.StringPointer("development"),
			}

			if _, err := store.Update(ctx, saved, upd); err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to update user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to update user.", tests.Success, testID)

			saved, err = store.QueryByEmail(ctx, *upd.Email)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user by Email : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve user by Email.", tests.Success, testID)

			diff := usr.DateUpdated.Sub(saved.DateUpdated)
			if diff > 0 {
				t.Fatalf("Should have a larger DateUpdated : sav %v, usr %v, dif %v", saved.DateUpdated, usr.DateUpdated, diff)
			}

			if saved.Name != *upd.Name {
				t.Errorf("\t%s\tTest %d:\tShould be able to see updates to Name.", tests.Failed, testID)
				t.Logf("\t\tTest %d:\tGot: %v", testID, saved.Name)
				t.Logf("\t\tTest %d:\tExp: %v", testID, *upd.Name)
			} else {
				t.Logf("\t%s\tTest %d:\tShould be able to see updates to Name.", tests.Success, testID)
			}

			if saved.Email != *upd.Email {
				t.Errorf("\t%s\tTest %d:\tShould be able to see updates to Email.", tests.Failed, testID)
				t.Logf("\t\tTest %d:\tGot: %v", testID, saved.Email)
				t.Logf("\t\tTest %d:\tExp: %v", testID, *upd.Email)
			} else {
				t.Logf("\t%s\tTest %d:\tShould be able to see updates to Email.", tests.Success, testID)
			}

			if saved.Department != *upd.Department {
				t.Errorf("\t%s\tTest %d:\tShould be able to see updates to Department.", tests.Failed, testID)
				t.Logf("\t\tTest %d:\tGot: %v", testID, saved.Department)
				t.Logf("\t\tTest %d:\tExp: %v", testID, *upd.Department)
			} else {
				t.Logf("\t%s\tTest %d:\tShould be able to see updates to Department.", tests.Success, testID)
			}

			claims := auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "Test",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
				Roles: []auth.Role{auth.Admin},
			}

			if err := store.Delete(ctx, claims, saved); err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to delete user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to delete user.", tests.Success, testID)

			_, err = store.QueryByID(ctx, saved.ID)
			if !errors.Is(err, user.ErrNotFound) {
				t.Fatalf("\t%s\tTest %d:\tShould NOT be able to retrieve user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould NOT be able to retrieve user.", tests.Success, testID)
		}
	}

}

func paging(t *testing.T) {
	stest := tests.NewTest(t, c)
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
			t.Error(string(debug.Stack()))
		}
		stest.Teardown()
	}()

	store := user.NewStore(stest.Log, stest.DB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Log("Given the need to page through User records.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen paging through 2 users.", testID)
		{

			name := "User Gopher"
			users1, err := store.Query(ctx, 1, 1)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user %q : %s.", tests.Failed, testID, name, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve user %q.", tests.Success, testID, name)

			if len(users1) != 1 && users1[0].Name == name {
				t.Fatalf("\t%s\tTest %d:\tShould have a single user for %q : %s.", tests.Failed, testID, name, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have a single user.", tests.Success, testID)

			name = "Admin Gopher"
			users2, err := store.Query(ctx, 1, 1)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user %q : %s.", tests.Failed, testID, name, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve users %q.", tests.Success, testID, name)

			if len(users2) != 1 && users2[0].Name == name {
				t.Fatalf("\t%s\tTest %d:\tShould have a single user for %q : %s.", tests.Failed, testID, name, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have a single user.", tests.Success, testID)

			users3, err := store.Query(ctx, 1, 2)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve 2 users for page 1 : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve 2 users for page 1.", tests.Success, testID)

			if len(users3) != 2 {
				t.Logf("\t\tTest %d:\tgot: %v", testID, len(users3))
				t.Logf("\t\tTest %d:\texp: %v", testID, 2)
				t.Fatalf("\t%s\tTest %d:\tShould have 2 users for page 1 : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have 2 users for page 1.", tests.Success, testID)

			if users3[0].ID == users3[1].ID {
				t.Logf("\t\tTest %d:\tUser1: %v", testID, users3[0].ID)
				t.Logf("\t\tTest %d:\tUser2: %v", testID, users3[1].ID)
				t.Fatalf("\t%s\tTest %d:\tShould have different users : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have different users.", tests.Success, testID)
		}
	}

}
