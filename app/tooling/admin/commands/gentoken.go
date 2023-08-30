package commands

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TODO(tcmhoang): Need to generlize the function to handle for every kid
func genToken() error {
	name := "zarf/keys/private.pem"
	file, err := os.Open(name)
	if err != nil {
		return err
	}

	privpem, err := io.ReadAll(io.LimitReader(file, 1024*1024))
	if err != nil {
		return fmt.Errorf("reading auth private key: %w", err)
	}

	privkey, err := jwt.ParseRSAPrivateKeyFromPEM(privpem)
	if err != nil {
		return fmt.Errorf("parsing auth private key: %w", err)
	}

	claims := struct {
		jwt.RegisteredClaims
		Roles []string
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "service project",
			Subject:   "123456789",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8760 * time.Hour).UTC()),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: []string{"ADMIN"},
	}

	method := jwt.GetSigningMethod("RS256")
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = "private"

	str, err := token.SignedString(privkey)
	if err != nil {
		return err
	}

	fmt.Println("TOKEN BEGIN")
	fmt.Println(str)
	fmt.Println("TOKEN END")

	fmt.Println()

	asn1Bs, err := x509.MarshalPKIXPublicKey(&privkey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	pubblk := pem.Block{
		Type:  "RSA PUBLICKEY",
		Bytes: asn1Bs,
	}

	if err := pem.Encode(os.Stdout, &pubblk); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	var parserClaims struct {
		jwt.RegisteredClaims
		Roles []string
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	getPubKeyFun := func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"]
		if !ok {
			return nil, errors.New("missing key id (kid) in token header")
		}
		kidID, ok := kid.(string)
		if !ok {
			return nil, errors.New("user token id (kid) must be string")
		}
		fmt.Println("KID", kidID)
		return &privkey.PublicKey, nil
	}

	parsedToken, err := parser.ParseWithClaims(str, &parserClaims, getPubKeyFun)

	if err != nil {
		return fmt.Errorf("parsing token: %w", err)
	}

	if !parsedToken.Valid {
		return fmt.Errorf("invalid Token")
	}

	fmt.Println("Token validate")

	return nil
}
