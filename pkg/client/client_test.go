package client

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"testing"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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
		Host:        os.Getenv("DEV_HOST"),
		BearerToken: os.Getenv("DEV_TOKEN"),
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

func TestList(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")

	result, err := c.ListResource("default", metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		}})
	check(t, err)
	fmt.Println(fmt.Sprintf("result: %+v %d", result.Object, len(result.Items)))
}

func TestCreate(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	body := []byte(`{"kind":"Namespace","apiVersion":"v1","metadata":{"name":"f9","creationTimestamp":null},"spec":{},"status":{}}`)
	var obj unstructured.Unstructured
	obj.UnmarshalJSON(body)
	_, err := c.CreateResource(&obj)
	check(t, err)

}

func TestPatch(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	body := []byte(`{"metadata":{"labels":{"f":"1"}}}`)

	result, err := c.GetResource("", "f9", metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	})
	check(t, err)
	_, err = c.PatchResource(result, body, types.MergePatchType)
	check(t, err)

}

/*func TestDeleteCollection(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	err := c.DeleteCollection("", &metav1.DeleteOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
	}, metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		LabelSelector: "f=1",
	})
	check(t, err)
}*/

func TestDelete(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	err := c.DeleteResourceByName("", "f9", &metav1.DeleteOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	})
	//assert(t, apiErrors.IsNotFound(err), true)
	check(t, err)
}

func TestGet(t *testing.T) {
	c, _ := NewKubeClient(getConfig(), "dev")
	result, err := c.GetResource("default", "kubernetes", metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	})
	check(t, err)
	fmt.Println(result)

	_, err = c.GetResource("default", "kubernetes-not-exist", metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	})
	assert(t, apiErrors.IsNotFound(err), true)

}
