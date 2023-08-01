package main

import (
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/tcmhoang/sservices/app/services/sales-api/handlers"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var build = "develop"

func main() {
	logger, err := initLogger()
	if err != nil {
		log.Printf("Error constructing logger: %s", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if err := run(logger); err != nil {
		logger.Errorw("Startup", "ERROR", err)
		os.Exit(1)
	}

}

func run(log *zap.SugaredLogger) error {

	if _, err := maxprocs.Set(); err != nil {
		return fmt.Errorf("maxprocs %w", err)
	}

	g := runtime.GOMAXPROCS(0)
	log.Infof("Starting service build[%s] CPU[%d]\n", build, g)
	defer log.Infoln("Service ended")

	// TODO: Need to figure out timeouts for http service
	cfg := struct {
		conf.Version
		Web struct {
			APIHOST         string        `conf:"default:0.0.0.0:3000"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
		}
	}{
		Version: conf.Version{
			SVN:  build,
			Desc: "TCMHOANG",
		},
	}

	confout, err := initConfig(&cfg)

	if err != nil {
		return err
	}

	if err == nil && confout == "" { // help option
		return nil
	}

	log.Infow("Startup", "CONFIG", confout)
	expvar.NewString("build").Set(build)

	log.Infow("Startup", "Status", "Debug router started", "host", cfg.Web.DebugHost)
	initDebugMux(log, cfg.Web.DebugHost)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Infoln("Stopping service")
	return nil
}

func initDebugMux(log *zap.SugaredLogger, host string) {
	debugMux := handlers.DebugStdLibMux()

	go func() {
		if err := http.ListenAndServe(host, debugMux); err != nil {
			log.Errorw("Shutdown", "status", "Debug router closed", "host", host, "ERROR", err)
		}
	}()

}

func initConfig(cfg interface{}) (string, error) {
	const prefix = "SALES"

	help, err := conf.ParseOSArgs(prefix, cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return "", nil
		}
		return "", fmt.Errorf("passing config: %w", err)
	}
	out, err := conf.String(cfg)
	if err != nil {
		return "", fmt.Errorf("Generating config for stdout: %w", err)
	}

	return out, nil
}

func initLogger() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	config.InitialFields = map[string]interface{}{
		"service": "SALES-API",
	}

	log, err := config.Build()
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
