package keystore

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

type KeyStore struct {
	lock  sync.RWMutex
	store map[string]*rsa.PrivateKey
}

func New() *KeyStore {
	return &KeyStore{
		store: make(map[string]*rsa.PrivateKey),
	}
}

func NewMap(store map[string]*rsa.PrivateKey) *KeyStore {
	return &KeyStore{
		store: store,
	}
}

func NewFS(fsys fs.FS) (*KeyStore, error) {
	ks := New()

	traverse := func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walkdir failure: %w", err)
		}

		if de.IsDir() {
			return nil
		}
		if path.Ext(p) != ".pem" {
			return nil
		}

		file, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("opening key file: %w", err)
		}
		defer file.Close()

		privKey, err := io.ReadAll(io.LimitReader(file, 1024*1024))
		if err != nil {
			return fmt.Errorf("reading auth private key: %w", err)
		}

		parsedPrivKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
		if err != nil {
			return fmt.Errorf("parsing auth private key: %w", err)
		}

		ks.store[strings.TrimSuffix(de.Name(), ".pem")] = parsedPrivKey

		return nil
	}

	if err := fs.WalkDir(fsys, ".", traverse); err != nil {
		return nil, fmt.Errorf("walking dir: %w", err)
	}

	return ks, nil
}

func (ks *KeyStore) Add(privateKey *rsa.PrivateKey, kid string) {
	ks.lock.Lock()
	defer ks.lock.Unlock()

	ks.store[kid] = privateKey
}

func (ks *KeyStore) Remove(kid string) {
	ks.lock.Lock()
	defer ks.lock.Unlock()

	delete(ks.store, kid)
}

func (ks *KeyStore) PrivateKey(kid string) (*rsa.PrivateKey, error) {
	ks.lock.Lock()
	ks.lock.Unlock()

	privKey, found := ks.store["kid"]
	if !found {
		return nil, errors.New("kid lookup failed")
	}

	return privKey, nil
}

func (ks *KeyStore) PublicKey(kid string) (*rsa.PublicKey, error) {

	privKey, err := ks.PrivateKey(kid)
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}
