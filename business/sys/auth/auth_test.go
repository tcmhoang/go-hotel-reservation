package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tcmhoang/sservices/business/sys/auth"
)

const (
	success = "\u2713"
	failed  = "\u2717"
)

func TestAuth(t *testing.T) {
	t.Log("Given the need to be able to authenticate and authorize access.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen handling a single user.", testID)
		{
			const keyID = "TEST"
			privkey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("\t%s\tTest: %d:\tShould be able to create private key: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest: %d:\tShould be able to create private key.", success, testID)

			a, err := auth.New(keyID, &keyStore{pk: privkey})
			if err != nil {
				t.Fatalf("\t%s\tTest: %d:\tShould be able to create an authenticator: %v", failed, testID, err)
			}

			claims := auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "test",
					Subject:   "TEST",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour).UTC()),
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
				},
				Roles: []auth.Roles{auth.Admin},
			}

			token, err := a.GeneratingToken(claims)
			if err != nil {
				t.Fatalf("\t%s\tTest: %d:\tShould be able to generate the claims: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest: %d:\tShould be able to generate the claims.", success, testID)

			parsedClaims, err := a.ValidateToken(token)
			if err != nil {
				t.Fatalf("\t%s\tTest: %d:\tShould be able to parse the claims: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest: %d:\tShould be able to parse the claims.", success, testID)

			if exp, got := len(claims.Roles), len(parsedClaims.Roles); exp != got {
				t.Logf("\t\tTest %d:\texp: %v", testID, exp)
				t.Logf("\t\tTest %d:\tgot: %v", testID, got)
				t.Fatalf("\t%s\tTest: %d:\tShould have the expected number of roles: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest: %d:\tShould have expected number of roles.", success, testID)

			if exp, got := claims.Roles[0], parsedClaims.Roles[0]; exp != got {
				t.Logf("\t\tTest %d:\texp: %v", testID, exp)
				t.Logf("\t\tTest %d:\tgot: %v", testID, got)
				t.Fatalf("\t%s\tTest: %d:\tShould have the expected roles: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest: %d:\tShould have he expected roles.", success, testID)

		}
	}
}

type keyStore struct {
	pk *rsa.PrivateKey
}

func (ks *keyStore) PrivateKey(kid string) (*rsa.PrivateKey, error) {
	return ks.pk, nil
}

func (ks *keyStore) PublicKey(kid string) (*rsa.PublicKey, error) {
	return &ks.pk.PublicKey, nil
}
