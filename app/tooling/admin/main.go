package main

import (
	"fmt"
	"os"

	"github.com/tcmhoang/sservices/foundation/logger"
	"go.uber.org/zap"
)

var build = "develop"

func main() {
	log, err := logger.New("ADMIN")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer log.Sync()

	if err := run(log); err != nil {
		os.Exit(1)
	}

}

func run(logger *zap.SugaredLogger) error {
	// TODO(tcmhoang): need to handle cmd passed-in to call the right functions
	return nil
}
