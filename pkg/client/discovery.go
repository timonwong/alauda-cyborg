package client

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/juju/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type APIGroupMap struct {
	// index by cluster
	M map[string]*metav1.APIGroupList
	sync.RWMutex
}

type APIResourceMap struct {
	// index by cluster
	M map[string][]*metav1.APIResourceList
	sync.RWMutex
}

var (
	allAPIGroupMap = APIGroupMap{
		M: make(map[string]*metav1.APIGroupList),
	}
	allAPIResourceMap = APIResourceMap{
		M: make(map[string][]*metav1.APIResourceList),
	}
)

func (c *KubeClient) GetGroupVersionList() (*metav1.APIGroupList, error) {
	gl, ok := allAPIGroupMap.M[c.cluster]
	if ok {
		return gl, nil
	}

	if err := c.syncGroupVersion(true); err != nil {
		return nil, err
	}
	gl, ok = allAPIGroupMap.M[c.cluster]
	if ok {
		return gl, nil
	}
	return nil, fmt.Errorf("find no group version list for %s", c.cluster)
}

// GetResourceList gets api resource list of the cluster.
func (c *KubeClient) GetApiResourceList() ([]*metav1.APIResourceList, error) {
	rl, ok := allAPIResourceMap.M[c.cluster]
	if ok {
		return rl, nil
	}

	if err := c.syncAPIResourceMap(true); err != nil {
		return nil, err
	}

	rl, ok = allAPIResourceMap.M[c.cluster]
	if ok {
		return rl, nil
	}
	return nil, fmt.Errorf("find no resource list for %s", c.cluster)
}

// GetResourceByKind gets the name of resource type by the resource kind.
// eg: Deployment -> deployments
func (c *KubeClient) GetResourceTypeByKind(kind string) (string, error) {
	r, err := c.GetApiResourceByKind(kind)
	if err != nil {
		return "", errors.Trace(
			ErrorResourceTypeNotFound{message: fmt.Sprintf("resource kind '%s' not found", kind)},
		)
	}
	return r.Name, nil
}

func IsSubResource(resource *metav1.APIResource) bool {
	return strings.Contains(resource.Name, "/")
}

func canResourceList(resource metav1.APIResource) bool {
	if strings.Contains(resource.Name, "/") {
		return false
	}

	for _, v := range resource.Verbs {
		if v == "list" {
			return true
		}
	}
	return false
}

// GetResourceByKind gets the APIResource by the resource kind， skip sub resources
func (c *KubeClient) GetApiResourceByKind(kind string) (*metav1.APIResource, error) {
	resources, err := c.GetApiResourceList()
	if err != nil {
		return nil, errors.Trace(ErrorResourceTypeNotFound{
			message: fmt.Sprintf("find apiResource for kind error: %s %s", c.cluster, kind)})
	}

	for _, rl := range resources {
		//TODO: test
		for _, r := range rl.APIResources {
			if r.Kind == kind && !IsSubResource(&r) {
				return &r, nil
			}
		}
	}
	return nil, errors.Trace(ErrorResourceTypeNotFound{
		message: fmt.Sprintf("find apiResource for kind error: %s %s", c.cluster, kind)})
}

// GetApiResourceByName gets APIResource by the resource type name and
// the preferred api version. If the preferredVersion not exist, the first
// available version will be returned.
func (c *KubeClient) GetApiResourceByName(name string, preferredVersion string) (*metav1.APIResource, error) {
	resources, err := c.GetApiResourceList()
	if err != nil {
		return nil, errors.Trace(NewTypeNotFoundError(fmt.Sprintf("find apiResource for name error: %s %s", c.cluster, name)))
	}

	var cans []*metav1.APIResource
	for _, rl := range resources {
		for idx, r := range rl.APIResources {
			if r.Name == name {
				cans = append(cans, &rl.APIResources[idx])
			}
		}
	}

	if len(cans) == 0 {
		return nil, errors.Trace(NewTypeNotFoundError(fmt.Sprintf("find apiResource for name error: %s %s", c.cluster, name)))

	}

	for _, item := range cans {
		gv := schema.GroupVersion{
			Group:   item.Group,
			Version: item.Version,
		}
		if gv.String() == preferredVersion {
			return item, nil
		}
	}

	return cans[0], nil

}

// GetVersionByGroup gets the preferred version of a group.
func (c *KubeClient) GetVersionByGroup(group string) (string, error) {
	data, err := c.GetGroupVersionList()
	if err != nil {
		return "", err
	}

	for _, item := range data.Groups {
		if item.Name == group {
			return item.PreferredVersion.Version, nil
		}
	}

	return "", errors.Trace(NewTypeNotFoundError(fmt.Sprintf("find version for group error: %s %s", c.cluster, group)))
}

// GetGroupVersionByName gets the group version of a resource by it's type name and
// the preferred api version.
func (c *KubeClient) GetGroupVersionByName(name string, preferredVersion string) (schema.GroupVersion, error) {
	apiRes, err := c.GetApiResourceByName(name, preferredVersion)
	if err != nil {
		return schema.GroupVersion{}, err
	}

	return schema.GroupVersion{
		Group:   apiRes.Group,
		Version: apiRes.Version,
	}, nil
}

// syncGroupVersion happens in client init and new resource added
// if force == false, skip sync if already have data in the map
func (c *KubeClient) syncGroupVersion(force bool) error {
	if !force {
		_, ok := allAPIResourceMap.M[c.cluster]
		if ok {
			return nil
		}
	}
	glog.Infof("force resync group version info for cluster: %s", c.cluster)
	groups, err := c.kc.Discovery().ServerGroups()
	if err != nil {
		return err
	}
	glog.Infof("resync group version info %+v for cluster: %s", groups, c.cluster)
	allAPIGroupMap.Lock()
	allAPIGroupMap.M[c.cluster] = groups
	allAPIGroupMap.Unlock()

	return nil
}

// syncKindResourceMap happens in client init and new resource added
// if force == false, skip sync if already have data in the map
func (c *KubeClient) syncAPIResourceMap(force bool) error {
	if !force {
		_, ok := allAPIResourceMap.M[c.cluster]
		if ok {
			return nil
		}
	}
	glog.Infof("force resync api resources for cluster: %s", c.cluster)
	serverResourceList, err := c.kc.Discovery().ServerResources()
	if err != nil {
		return err
	}
	glog.Infof("resync api resource info %+v for cluster: %s", serverResourceList, c.cluster)

	// set group and version
	for _, rl := range serverResourceList {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			glog.Errorf("parse group version for %s error: %s", rl.GroupVersion, err)
			continue
		}
		for idx := range rl.APIResources {
			rl.APIResources[idx].Group = gv.Group
			rl.APIResources[idx].Version = gv.Version
		}
	}

	allAPIResourceMap.Lock()
	allAPIResourceMap.M[c.cluster] = serverResourceList
	allAPIResourceMap.Unlock()
	return nil
}

// ConfigForResource generates the REST config of k8s client for the resource type and version
func (c *KubeClient) ConfigForResource(name string, preferredVersion string) (rest.Config, error) {
	newCfg := *c.config
	gv, err := c.GetGroupVersionByName(name, preferredVersion)

	glog.Infof("Found gv %s for %s", gv.String(), name)
	if err != nil {
		return newCfg, err
	}

	newCfg.GroupVersion = &gv
	newCfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	switch newCfg.GroupVersion.Group {
	case "":
		newCfg.APIPath = "/api"
	default:
		newCfg.APIPath = "/apis"
	}
	return newCfg, nil
}

func (c *KubeClient) IsClusterScopeResource(kind string) bool {
	r, err := c.GetApiResourceByKind(kind)
	if err != nil {
		// Default as no cluster scope
		return false
	}

	return !r.Namespaced
}
