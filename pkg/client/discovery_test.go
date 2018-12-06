package client

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/juju/errors"
)

func TestSync(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	//t.Logf("APIResource Map: %+v", allAPIResourceMap.M)
	//t.Logf("GroupVersion Map: %+v", allAPIGroupMap.M)
	res, err := c.GetApiResourceByKind("Deployment")
	check(t, err)
	t.Logf("Deployment: %+v", res)

	res, err = c.GetApiResourceByName("deployments", "")
	check(t, err)
	t.Logf("Deployment: %+v", res)

	res, err = c.GetApiResourceByName("deployments", "apps/v1")
	check(t, err)
	t.Logf("Deployment: %+v", res)
}

func TestGetByName(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	r, err := c.GetApiResourceByName("services", "")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if r.Kind != "Service" {
		t.Errorf("Expect 'Service', got '%s'", r.Kind)
		return
	}

	r, err = c.GetApiResourceByName("service", "")
	fmt.Println(err.Error())
	if !IsResourceTypeNotFound(err) {
		t.Errorf("Expect ErrorResourceTypeNotFind, got %s", reflect.TypeOf(errors.Cause(err)))
	}
}
