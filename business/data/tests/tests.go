// Package tests contains supporting code for running tests that hit the DB.
package tests

import (
	"bytes"
	"context"
	"crypto/rsa"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/business/data/schema"
	"github.com/tcmhoang/sservices/business/data/store/user"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/sys/database"
	"github.com/tcmhoang/sservices/foundation/docker"
	"github.com/tcmhoang/sservices/foundation/logger"
	"go.uber.org/zap"
)

const (
	Success = "\u2713"
	Failed  = "\u2717"
)

func InitDB() (*docker.Container, error) {
	image := "postgres:15.4"
	port := "5432"
	dockerArgs := []string{"-e", "POSTGRES_PASSWORD=postgres"}
	appArgs := []string{}

	pc, err := docker.InitContainer(image, port, dockerArgs, appArgs)
	if err != nil {
		return nil, fmt.Errorf("DB initialize: %w", err)
	}

	fmt.Printf("Image:       %s\n", image)
	fmt.Printf("ContainerID: %s\n", pc.ID)
	fmt.Printf("Host:        %s\n", pc.Host)
	return pc, nil
}

func StopDB(c *docker.Container) {
	docker.StopContainer(c.ID)
	fmt.Println("Stopped:", c.ID)
}

type State struct {
	DB       *sqlx.DB
	Log      *zap.SugaredLogger
	Auth     *auth.Auth
	Teardown func()
	t        *testing.T
}

func NewTest(t *testing.T, c *docker.Container) *State {
	r, w, _ := os.Pipe()

	old := os.Stdout
	os.Stdout = w

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbinitial, err := database.Open(database.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       c.Host,
		Name:       "postgres",
		DisableTLS: true,
	})

	if err != nil {
		t.Fatalf("Opening database connection: %v", err)
	}

	t.Log("Waiting for database to be ready ...")

	if err := database.StatusCheck(ctx, dbinitial); err != nil {
		t.Fatalf("status check database: %v", err)
	}

	if _, err := dbinitial.ExecContext(context.Background(), "CREATE DATABASE test"); err != nil {
		t.Fatalf("creating database test: %v", err)
	}
	dbinitial.Close()

	t.Log("Database ready")

	dbconn, err := database.Open(database.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       c.Host,
		Name:       "test",
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("Opening database connection: %v", err)
	}

	t.Log("Migrate and seed database ...")

	if err := schema.Migrate(ctx, dbconn); err != nil {
		t.Logf("Logs for %s\n%s:", c.ID, docker.DumpContainerLogs(c.ID))
		t.Fatalf("Migrating error: %s", err)
	}

	if err := schema.Seed(ctx, dbconn); err != nil {
		t.Logf("Logs for %s\n%s:", c.ID, docker.DumpContainerLogs(c.ID))
		t.Fatalf("Seeding error: %s", err)
	}

	log, err := logger.New("SALES-API")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer log.Sync()

	if err != nil {
		t.Fatalf("Initializing logger: %v", err)
	}

	t.Log("Ready for testing ...")

	teardown := func() {
		t.Helper()
		dbconn.Close()
		docker.StopContainer(c.ID)

		log.Sync()

		w.Close()

		var buf bytes.Buffer
		io.Copy(&buf, r)
		os.Stdout = old

		fmt.Println("******************** LOGS ********************")
		fmt.Print(buf.String())
		fmt.Println("******************** LOGS ********************")

	}

	a, err := auth.New(kid, &keyStore{})

	if err != nil {
		t.Fatal(err)
	}

	test := State{
		DB:       dbconn,
		Log:      log,
		Auth:     a,
		Teardown: teardown,
		t:        t,
	}

	return &test
}

func (s *State) Token(email string, pass string) string {
	s.t.Log("Generating token for test ...")

	addr, _ := mail.ParseAddress(email)

	store := user.NewStore(s.Log, s.DB)
	user, err := store.QueryByEmail(context.Background(), *addr)
	if err != nil {
		return ""
	}

	var roles []auth.Role

	for _, r := range user.Roles {
		switch strings.ToLower(r) {
		case "admin":
			roles = append(roles, auth.Admin)
		case "user":
			roles = append(roles, auth.User)

		default:

		}
	}
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    "service project",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: roles,
	}

	token, err := s.Auth.GenerateToken(claims)
	if err != nil {
		s.t.Fatal(err)
	}

	return token
}

func StringPointer(s string) *string {
	return &s
}

type keyStore struct{}

func (ks *keyStore) PrivateKey(kid string) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
}

func (ks *keyStore) PublicKey(kid string) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
}

const (
	kid = "s4sKIjD9kIRjxs2tulPqGLdxSfgPErRN1Mu3Hd9k9NQ"

	privateKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAvMAHb0IoLvoYuW2kA+LTmnk+hfnBq1eYIh4CT/rMPCxgtzjq
U0guQOMnLg69ydyA5uu37v6rbS1+stuBTEiMQl/bxAhgLkGrUhgpZ10Bt6GzSEgw
QNloZoGaxe4p20wMPpT4kcMKNHkQds3uONNcLxPUmfjbbH64g+seg28pbgQPwKFK
tF7bIsOBgz0g5Ptn5mrkdzqMPUSy9k9VCu+R42LH9c75JsRzz4FeN+VzwMAL6yQn
ZvOi7/zOgNyxeVia8XVKykrnhgcpiOn5oaLRBzQGN00Z7TuBRIfDJWU21qQN4Cq7
keZmMP4gqCVWjYneK4bzrG/+H2w9BJ2TsmMGvwIDAQABAoIBAFQmQKpHkmavNYql
6POaksBRwaA1YzSijr7XJizGIXvKRSwqgb2zdnuTSgpspAx09Dr/aDdy7rZ0DAJt
fk2mInINDottOIQm3txwzTS58GQQAT/+fxTKWJMqwPfxYFPWqbbU76T8kXYna0Gs
OcK36GdMrgIfQqQyMs0Na8MpMg1LmkAxuqnFCXS/NMyKl9jInaaTS+Kz+BSzUMGQ
zebfLFsf2N7sLZuimt9zlRG30JJTfBlB04xsYMo734usA2ITe8U0XqG6Og0qc6ev
6lsoM8hpvEUsQLcjQQ5up7xx3S2stZJ8o0X8GEX5qUMaomil8mZ7X5xOlEqf7p+v
lXQ46cECgYEA2lbZQON6l3ZV9PCn9j1rEGaXio3SrAdTyWK3D1HF+/lEjClhMkfC
XrECOZYj+fiI9n+YpSog+tTDF7FTLf7VP21d2gnhQN6KAXUnLIypzXxodcC6h+8M
ZGJh/EydLvC7nPNoaXx96bohxzS8hrOlOlkCbr+8gPYKf8qkbe7HyxECgYEA3U6e
x9g4FfTvI5MGrhp2BIzoRSn7HlNQzjJ71iMHmM2kBm7TsER8Co1PmPDrP8K/UyGU
Q25usTsPSrHtKQEV6EsWKaP/6p2Q82sDkT9bZlV+OjRvOfpdO5rP6Q95vUmMGWJ/
S6oimbXXL8p3gDafw3vC1PCAhoaxMnGyKuZwlM8CgYEAixT1sXr2dZMg8DV4mMfI
8pqXf+AVyhWkzsz+FVkeyAKiIrKdQp0peI5C/5HfevVRscvX3aY3efCcEfSYKt2A
07WEKkdO4LahrIoHGT7FT6snE5NgfwTMnQl6p2/aVLNun20CHuf5gTBbIf069odr
Af7/KLMkjfWs/HiGQ6zuQjECgYEAv+DIvlDz3+Wr6dYyNoXuyWc6g60wc0ydhQo0
YKeikJPLoWA53lyih6uZ1escrP23UOaOXCDFjJi+W28FR0YProZbwuLUoqDW6pZg
U3DxWDrL5L9NqKEwcNt7ZIDsdnfsJp5F7F6o/UiyOFd9YQb7YkxN0r5rUTg7Lpdx
eMyv0/UCgYEAhX9MPzmTO4+N8naGFof1o8YP97pZj0HkEvM0hTaeAQFKJiwX5ijQ
xumKGh//G0AYsjqP02ItzOm2mWnbI3FrNlKmGFvR6VxIZMOyXvpLofHucjJ5SWli
eYjPklKcXaMftt1FVO4n+EKj1k1+Tv14nytq/J5WN+r4FBlNEYj/6vg=
-----END RSA PRIVATE KEY-----
`
	publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvMAHb0IoLvoYuW2kA+LT
mnk+hfnBq1eYIh4CT/rMPCxgtzjqU0guQOMnLg69ydyA5uu37v6rbS1+stuBTEiM
Ql/bxAhgLkGrUhgpZ10Bt6GzSEgwQNloZoGaxe4p20wMPpT4kcMKNHkQds3uONNc
LxPUmfjbbH64g+seg28pbgQPwKFKtF7bIsOBgz0g5Ptn5mrkdzqMPUSy9k9VCu+R
42LH9c75JsRzz4FeN+VzwMAL6yQnZvOi7/zOgNyxeVia8XVKykrnhgcpiOn5oaLR
BzQGN00Z7TuBRIfDJWU21qQN4Cq7keZmMP4gqCVWjYneK4bzrG/+H2w9BJ2TsmMG
vwIDAQAB
-----END PUBLIC KEY-----`
)
