package client

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type KubeClient struct {
	cluster    string
	config     *rest.Config
	kc         kubernetes.Interface
	clientPool dynamic.ClientPool
}

func NewKubeClient(cfg *rest.Config, cluster string) (*KubeClient, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := KubeClient{
		cluster:    cluster,
		config:     cfg,
		kc:         kc,
		clientPool: dynamic.NewDynamicClientPool(cfg),
	}
	if err := c.syncGroupVersion(false); err != nil {
		return nil, err
	}
	if err := c.syncKindResourceMap(false); err != nil {
		return nil, err
	}
	return &c, nil
}

// getClient get client from unstructured
func (c *KubeClient) getClient(resource *unstructured.Unstructured) (dynamic.Interface, error) {
	return c.getClientByGVK(resource.GroupVersionKind())
}

func (c *KubeClient) getClientByGVK(gvk schema.GroupVersionKind) (dynamic.Interface, error) {
	return c.clientPool.ClientForGroupVersionKind(gvk)
}

func (c *KubeClient) deleteResource(resource *unstructured.Unstructured) error {
	client, err := c.getClient(resource)
	if err != nil {
		return err
	}

	apiResource, err := c.GetApiResourceByKind(resource.GetKind())
	if err != nil {
		return err
	}

	return client.Resource(apiResource, resource.GetNamespace()).Delete(resource.GetName(), &metav1.DeleteOptions{})

}

func (c *KubeClient) createResource(resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	client, err := c.getClient(resource)
	if err != nil {
		return nil, err
	}

	apiResource, err := c.GetApiResourceByKind(resource.GetKind())
	if err != nil {
		return nil, err
	}

	return client.Resource(apiResource, resource.GetNamespace()).Create(resource)

}

func (c *KubeClient) updateResource(resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	client, err := c.getClient(resource)
	if err != nil {
		return nil, err
	}

	apiResource, err := c.GetApiResourceByKind(resource.GetKind())
	if err != nil {
		return nil, err
	}

	return client.Resource(apiResource, resource.GetNamespace()).Update(resource)
}

func (c *KubeClient) patchResource(resource *unstructured.Unstructured, body []byte, jt types.PatchType) (*unstructured.Unstructured, error) {
	client, err := c.getClient(resource)
	if err != nil {
		return nil, err
	}

	apiResource, err := c.GetApiResourceByKind(resource.GetKind())
	if err != nil {
		return nil, err
	}

	return client.Resource(apiResource, resource.GetNamespace()).Patch(resource.GetName(), jt, body)
}
