package client

import (
	"k8s.io/client-go/rest"
	"testing"
)

func assert(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
func check(t *testing.T, err error) {
	if err != nil {
		t.Errorf("get error: %+v", err)
	}
}

func getConfig() *rest.Config {
	return &rest.Config{
		Host:        "https://119.28.224.65:6443",
		BearerToken: "fuckfk.baidu12345678910",
		APIPath:     "apis",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
}

func TestGetClient(t *testing.T) {
	c, err := NewKubeClient(getConfig(), "dev")
	if err != nil {
		t.Errorf("get client error: %s", err.Error())
	}

	version, err := c.GetVersionByGroup("apps")
	check(t, err)
	assert(t, version, "v1")

}
