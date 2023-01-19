//go:build envtest
// +build envtest

package client

import (
	"fmt"
	"os"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testingCfg *rest.Config
)

func getConfig() *rest.Config {
	return testingCfg
}

func TestMain(m *testing.M) {
	var err error
	fmt.Println("bootstrapping test environment")
	testEnv := &envtest.Environment{
		ControlPlaneStartTimeout: time.Second * 60,
	}
	testingCfg, err = testEnv.Start()
	if err != nil {
		panic(err)
	}

	var exitCode int
	func() {
		defer func() {
			fmt.Println("tearing down the test environment")
			if err := testEnv.Stop(); err != nil {
				panic(err)
			}
		}()

		exitCode = m.Run()
	}()
	os.Exit(exitCode)
}
