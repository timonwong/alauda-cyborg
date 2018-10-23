package client

import (
	"fmt"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"github.com/golang/glog"
)

// APIGroupMap indexed by group
type APIGroupMap map[string]metav1.APIGroup

// APIResourceVersionMap defines an APIResource map type using apiVersion as key.
type APIResourceVersionMap map[string]metav1.APIResource

// APIResourceMap is api resources indexed by resource kind(Deployment,Service....)
type APIResourceMap map[string]APIResourceVersionMap

type GroupVersionMap struct {
	M map[string]APIGroupMap
	sync.RWMutex
}

type KindResourceMap struct {
	// index by cluster
	M map[string]APIResourceMap
	// API resource map indexed by resource type name
	MByName map[string]APIResourceMap
	sync.RWMutex
}

var AllGroupVersion = GroupVersionMap{
	M: make(map[string]APIGroupMap),
}

var AllKindResourceMap = KindResourceMap{
	M:       make(map[string]APIResourceMap),
	MByName: make(map[string]APIResourceMap),
}

// AllAPIResourceMap stores all api resource lists of k8s clusters.
var AllAPIResourceMap = make(map[string][]*metav1.APIResourceList)

type ErrorResourceKindNotFound struct {
	kind string
}

func (e ErrorResourceKindNotFound) Error() string {
	return fmt.Sprintf("resource kind '%s' not found", e.kind)
}

func IsResourceKindNotFound(err error) bool {
	_, ok := err.(ErrorResourceKindNotFound)
	return ok
}

// GetResourceList gets api resource list of the cluster.
func (c *KubeClient) GetResourceList() ([]*metav1.APIResourceList, error) {
	rl, ok := AllAPIResourceMap[c.cluster]
	if ok {
		return rl, nil
	}

	if err := c.syncKindResourceMap(true); err != nil {
		return nil, err
	}

	rl, ok = AllAPIResourceMap[c.cluster]
	if ok {
		return rl, nil
	}
	return nil, fmt.Errorf("find no resource list for %s", c.cluster)
}

// GetResourceByKind gets the name of resource type by the resource kind.
// eg: Deployment -> deployments
func (c *KubeClient) GetResourceByKind(kind string) (string, error) {
	r, err := c.GetApiResourceByKind(kind)
	if err != nil {
		return "", ErrorResourceKindNotFound{kind: kind}
	}
	return r.Name, nil
}

// GetResourceByKind gets the APIResource by the resource kind.
func (c *KubeClient) GetApiResourceByKind(kind string) (*metav1.APIResource, error) {
	getAPIResource := func() (*metav1.APIResource, error) {
		data, ok := AllKindResourceMap.M[c.cluster]
		if !ok {
			return nil, fmt.Errorf("api resource map cache not init for cluster: %s", c.cluster)
		}

		r, ok := data[kind]
		if ok {
			for _, v := range r {
				return &v, nil
			}
		}
		return nil, fmt.Errorf("find apiResource for kind error: %s %s", c.cluster, kind)
	}

	r, err := getAPIResource()
	if err == nil {
		return r, nil
	}

	if err := c.syncKindResourceMap(true); err != nil {
		return nil, err
	}

	return getAPIResource()
}

// GetApiResourceByName gets APIResource by the resource type name and
// the preferred api version. If the preferredVersion not exist, the first
// available version will be returned.
func (c *KubeClient) GetApiResourceByName(name string, preferredVersion string) (*metav1.APIResource, error) {
	getAPIResource := func() (*metav1.APIResource, error) {
		data, ok := AllKindResourceMap.MByName[c.cluster]
		if !ok {
			return nil, fmt.Errorf("api resource map cache not init for cluster: %s", c.cluster)
		}

		r, ok := data[name]
		if ok {
			v, exist := r[preferredVersion]
			if exist {
				return &v, nil
			}
			for _, v = range r {
				return &v, nil
			}
		}
		return nil, fmt.Errorf("find apiResource for name error: %s %s", c.cluster, name)
	}

	r, err := getAPIResource()
	if err == nil {
		return r, nil
	}

	if err := c.syncKindResourceMap(true); err != nil {
		return nil, err
	}

	return getAPIResource()
}

// GetVersionByGroup gets the preferred version of a group.
func (c *KubeClient) GetVersionByGroup(group string) (string, error) {
	data, ok := AllGroupVersion.M[c.cluster]
	if !ok {
		return "", fmt.Errorf("group verion map cache not init for cluster: %s", c.cluster)
	}

	grp, ok := data[group]
	if ok {
		return grp.PreferredVersion.Version, nil
	} else {
		if err := c.syncGroupVersion(true); err != nil {
			return "", err
		}
	}

	grp, ok = data[group]
	if ok {
		return grp.PreferredVersion.Version, nil
	}
	return "", fmt.Errorf("find version for group error: %s %s", c.cluster, group)

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
		_, ok := AllGroupVersion.M[c.cluster]
		if ok {
			return nil
		}
	}
	glog.Infof("force resync group version info for cluster: %s", c.cluster)
	groups, err := c.kc.Discovery().ServerGroups()
	if err != nil {
		return err
	}

	m := make(APIGroupMap)

	for _, item := range groups.Groups {
		m[item.Name] = item
	}
	glog.Infof("resync group version info %+v for cluster: %s", m, c.cluster)
	AllGroupVersion.Lock()
	AllGroupVersion.M[c.cluster] = m
	AllGroupVersion.Unlock()

	return nil
}

// syncKindResourceMap happens in client init and new resource added
// if force == false, skip sync if already have data in the map
func (c *KubeClient) syncKindResourceMap(force bool) error {
	if !force {
		_, ok := AllKindResourceMap.M[c.cluster]
		if ok {
			return nil
		}
	}
	glog.Infof("force resync api resources for cluster: %s", c.cluster)
	serverResourceList, err := c.kc.Discovery().ServerResources()
	if err != nil {
		return err
	}
	m := make(APIResourceMap)
	mByName := make(APIResourceMap)

	for _, rl := range serverResourceList {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			glog.Errorf("parse group version for %s error: %s", rl.GroupVersion, err)
			continue
		}
		for _, r := range rl.APIResources {
			if canResourceList(r) {
				if m[r.Kind] == nil {
					m[r.Kind] = make(APIResourceVersionMap)
				}
				// originally empty
				r.Group = gv.Group
				r.Version = gv.Version

				m[r.Kind][gv.String()] = r
				mByName[r.Name] = m[r.Kind]
			}

		}
	}

	AllKindResourceMap.Lock()
	AllAPIResourceMap[c.cluster] = serverResourceList
	AllKindResourceMap.M[c.cluster] = m
	AllKindResourceMap.MByName[c.cluster] = mByName
	AllKindResourceMap.Unlock()
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

	return r.Namespaced
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
