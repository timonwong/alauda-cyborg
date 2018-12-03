package client

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

func (c *KubeClient) DeleteResource(resource *unstructured.Unstructured) error {
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

func (c *KubeClient) CreateResource(resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
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

func (c *KubeClient) UpdateResource(resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
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

func (c *KubeClient) GetResource(namespace, name string, options metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvk := schema.FromAPIVersionAndKind(options.APIVersion, options.Kind)
	client, err := c.getClientByGVK(gvk)
	if err != nil {
		return nil, err
	}

	ar, err := c.GetApiResourceByKind(gvk.Kind)
	if err != nil {
		return nil, err
	}

	return client.Resource(ar, namespace).Get(name, options)
}

func (c *KubeClient) ListResource(namespace string, options metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	gvk := schema.FromAPIVersionAndKind(options.APIVersion, options.Kind)
	client, err := c.getClientByGVK(gvk)
	if err != nil {
		return nil, err
	}

	ar, err := c.GetApiResourceByKind(gvk.Kind)
	if err != nil {
		return nil, err
	}

	object, err := client.Resource(ar, namespace).List(options)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	var ul unstructured.UnstructuredList
	if err := ul.UnmarshalJSON(bytes); err != nil {
		return nil, err
	}

	return &ul, nil
}

func (c *KubeClient) PatchResource(resource *unstructured.Unstructured, body []byte, jt types.PatchType) (*unstructured.Unstructured, error) {
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
