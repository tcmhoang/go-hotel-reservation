package tests

import (
	"fmt"
	"testing"

	"github.com/tcmhoang/sservices/business/data/tests"
	"github.com/tcmhoang/sservices/foundation/docker"
)

var c *docker.Container

func TestMain(m *testing.M) {
	var err error
	c, err = tests.InitDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tests.StopDB(c)

	m.Run()
}
