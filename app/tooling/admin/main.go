package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	err := genToken()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

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

func genKey() error {
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return err
	}

	privfile, err := os.Create("private.pem")
	if err != nil {
		return fmt.Errorf("creating private file: %w", err)
	}
	defer privfile.Close()

	privblk := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privkey),
	}

	if err := pem.Encode(privfile, &privblk); err != nil {
		return fmt.Errorf("encoding to private file: %w", err)
	}

	asn1Bs, err := x509.MarshalPKIXPublicKey(&privkey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	pubfile, err := os.Create("public.pem")
	if err != nil {
		return fmt.Errorf("creating public file: %w", err)
	}
	defer pubfile.Close()

	pubblk := pem.Block{
		Type:  "RSA PUBLICKEY",
		Bytes: asn1Bs,
	}

	if err := pem.Encode(pubfile, &pubblk); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	return nil
}
