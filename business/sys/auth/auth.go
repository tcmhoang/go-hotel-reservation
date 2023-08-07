package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type KeyLookup interface {
	PrivateKey(kid string) (*rsa.PrivateKey, error)
	PublicKey(kid string) (*rsa.PublicKey, error)
}

type Auth struct {
	activeKID string
	KeyLookup
	method  jwt.SigningMethod
	keyFunc func(t *jwt.Token) (interface{}, error)
	parser  *jwt.Parser
}

func New(activeKID string, kup KeyLookup) (*Auth, error) {
	if _, err := kup.PrivateKey(activeKID); err != nil {
		return nil, errors.New("active KID doesn't exist in store")
	}

	method := jwt.GetSigningMethod(jwt.SigningMethodRS256.Name)
	if method == nil {
		return nil, errors.New("configuring algorithm RS256")
	}

	keyfunc := func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"]
		if ok {
			return nil, errors.New("missing key id (kid) in the header token")
		}
		kidID, ok := kid.(string)
		if ok {
			return nil, errors.New("kid must be a string")
		}
		return kup.PublicKey(kidID)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))

	out := Auth{
		activeKID: activeKID,
		KeyLookup: kup,
		method:    method,
		keyFunc:   keyfunc,
		parser:    parser,
	}
	return &out, nil
}

func (a *Auth) GeneratingToken(c Claims) (string, error) {
	token := jwt.NewWithClaims(a.method, c)
	token.Header["kid"] = a.activeKID

	privkey, err := a.KeyLookup.PrivateKey(a.activeKID)
	if err != nil {
		return "", errors.New("kid lookup private failed")
	}

	mtoken, err := token.SignedString(privkey)
	if err != nil {
		return "", fmt.Errorf("siging token: %w", err)
	}

	return mtoken, nil
}

func (a *Auth) ValidateToken(tokenstr string) (Claims, error) {
	var claims Claims

	token, err := a.parser.ParseWithClaims(tokenstr, &claims, a.keyFunc)
	if err != nil {
		return Claims{}, fmt.Errorf("parsing tokem: %w", err)
	}

	if !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	return claims, nil
}
