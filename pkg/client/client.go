package client

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/dynamic"
)


type KubeClient struct {
	kc kubernets.Interface
	clientPool dynamic.ClientPool
}