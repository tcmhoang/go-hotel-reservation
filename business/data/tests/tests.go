// Package dbtest contains supporting code for running tests that hit the DB.
package data_tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/business/data/schema"
	"github.com/tcmhoang/sservices/business/sys/database"
	"github.com/tcmhoang/sservices/foundation/docker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	config.InitialFields = map[string]interface{}{
		"service": "TEST",
	}

	log, err := config.Build()

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

	test := State{
		DB:       dbconn,
		Log:      log.Sugar(),
		Teardown: teardown,
		t:        t,
	}

	return &test
}
