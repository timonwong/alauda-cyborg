package client

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubeClient struct {
	cluster string
	config  *rest.Config
	kc      kubernetes.Interface
}

func NewKubeClient(cfg *rest.Config, cluster string) (*KubeClient, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	c := KubeClient{
		cluster: cluster,
		config:  cfg,
		kc:      kc,
	}
	if err := c.syncGroupVersion(false); err != nil {
		return nil, err
	}
	if err := c.syncAPIResourceMap(false); err != nil {
		return nil, err
	}
	return &c, nil
}
