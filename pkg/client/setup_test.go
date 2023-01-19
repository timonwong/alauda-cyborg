//go:build !envtest
// +build !envtest

package client

import (
	"os"

	"k8s.io/client-go/rest"
)

func getConfig() *rest.Config {
	return &rest.Config{
		Host:        os.Getenv("DEV_HOST"),
		BearerToken: os.Getenv("DEV_TOKEN"),
		APIPath:     "apis",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
}
