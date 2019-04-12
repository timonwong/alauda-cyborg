package client

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"github.com/juju/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestSync(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	//t.Logf("APIResource Map: %+v", allAPIResourceMap.M)
	//t.Logf("GroupVersion Map: %+v", allAPIGroupMap.M)
	res, err := c.GetApiResourceByKind("Deployment")
	check(t, err)
	t.Logf("Deployment: %+v", res)

	res, err = c.GetApiResourceByKindInsensitive("deployment")
	check(t, err)
	t.Logf("deployment: %+v", res)

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

func TestIsNamespaceScoped(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	r, err := c.IsNamespaceScoped("deployments")
	check(t, err)
	assert(t, r, true)

	r, err = c.IsNamespaceScoped("clusterroles")
	check(t, err)
	assert(t, r, false)
}

func TestGetDynamicClient(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	dc, err := c.DynamicClientForResource("services", "")
	check(t, err)
	result, err := dc.List(metav1.ListOptions{})
	check(t, err)
	t.Logf("total services: %d", len(result.Items))

	ndc := dc.Namespace("default")
	result, err = ndc.List(metav1.ListOptions{})
	check(t, err)
	t.Logf("services in default ns : %d", len(result.Items))

	dc, err = c.DynamicClientForResource("persistentvolumes", "")
	check(t, err)
	result, err = dc.List(metav1.ListOptions{})
	check(t, err)
	pvs := len(result.Items)
	t.Logf("total pv: %d", pvs)
	// test if set namespace not work
	ndc = dc.Namespace("default")
	result, err = ndc.List(metav1.ListOptions{})

	dc, err = c.ClientForGVK(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolume",
	})
	check(t, err)
	result, err = dc.List(metav1.ListOptions{})
	assert(t, len(result.Items) == pvs, true)

}
