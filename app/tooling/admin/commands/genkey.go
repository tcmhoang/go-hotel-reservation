package commands

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func GenKey() error {
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

	fmt.Println("private and public key files generated")

	return nil
}
