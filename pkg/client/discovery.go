package client

import (
	"fmt"
	"strings"
	"sync"

	"github.com/juju/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
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

// GetResourceTypeByGroupKind gets the name of resource type by the resource Groupkind.
// eg: Deployment -> deployments
func (c *KubeClient) GetResourceTypeByGroupKind(gk metav1.GroupKind) (string, error) {
	r, err := c.GetApiResourceByGroupKind(gk)
	if err != nil {
		return "", errors.Trace(
			ErrorResourceTypeNotFound{message: fmt.Sprintf("resource kind '%s/%s' not found", gk.Group, gk.Kind)},
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

// GetApiResourceByKind get api resource by kind
func (c *KubeClient) GetApiResourceByKind(kind string) (*metav1.APIResource, error) {
	resource, err := c.getApiResourceByKind(kind, false)
	if err != nil {
		if IsResourceTypeNotFound(err) {
			// force resync and retry
			if err := c.syncAPIResourceMap(true); err != nil {
				return nil, err
			}
			return c.getApiResourceByKind(kind, false)
		}
	}
	return resource, err
}

// GetApiResourceByGroupKind get api resource by GroupKind
func (c *KubeClient) GetApiResourceByGroupKind(gk metav1.GroupKind) (*metav1.APIResource, error) {
	resource, err := c.getApiResourceByGroupKind(gk, false)
	if err != nil {
		if IsResourceTypeNotFound(err) {
			// force resync and retry
			if err := c.syncAPIResourceMap(true); err != nil {
				return nil, err
			}
			return c.getApiResourceByGroupKind(gk, false)
		}
	}
	return resource, err
}

// GetApiResourceByKindInsensitive get api resource by kind, but ignore case when compare
func (c *KubeClient) GetApiResourceByKindInsensitive(kind string) (*metav1.APIResource, error) {
	resource, err := c.getApiResourceByKind(kind, true)
	if err != nil {
		if IsResourceTypeNotFound(err) {
			// force resync and retry
			if err := c.syncAPIResourceMap(true); err != nil {
				return nil, err
			}
			return c.getApiResourceByKind(kind, true)
		}
	}
	return resource, err
}

// GetResourceByKind gets the APIResource by the resource kind， skip sub resources
func (c *KubeClient) getApiResourceByKind(kind string, ignoreCase bool) (*metav1.APIResource, error) {
	resources, err := c.GetApiResourceList()
	if err != nil {
		return nil, errors.Trace(ErrorResourceTypeNotFound{
			message: fmt.Sprintf("find apiResource for kind error: %s %s", c.cluster, kind),
		})
	}

	for _, rl := range resources {
		// TODO: test
		for _, r := range rl.APIResources {
			if !IsSubResource(&r) {
				if r.Kind == kind || (ignoreCase && strings.EqualFold(r.Kind, kind)) {
					return &r, nil
				}
			}
		}
	}
	return nil, errors.Trace(ErrorResourceTypeNotFound{
		message: fmt.Sprintf("find apiResource for kind error: %s %s", c.cluster, kind),
	})
}

// getApiResourceByGroupKind gets the APIResource by the resource GroupKind， skip sub resources
func (c *KubeClient) getApiResourceByGroupKind(gk metav1.GroupKind, ignoreCase bool) (*metav1.APIResource, error) {
	resources, err := c.GetApiResourceList()
	if err != nil {
		return nil, errors.Trace(ErrorResourceTypeNotFound{
			message: fmt.Sprintf("find apiResource for kind error: %s %s/%s", c.cluster, gk.Group, gk.Kind),
		})
	}

	for _, rl := range resources {
		// TODO: test
		for _, r := range rl.APIResources {
			if !IsSubResource(&r) {
				if (r.Kind == gk.Kind || (ignoreCase && strings.EqualFold(r.Kind, gk.Kind))) && r.Group == gk.Group {
					return &r, nil
				}
			}
		}
	}
	return nil, errors.Trace(ErrorResourceTypeNotFound{
		message: fmt.Sprintf("find apiResource for kind error: %s %s/%s", c.cluster, gk.Group, gk.Kind),
	})
}

// GetApiResourceByName gets APIResource by the resource type name and
// the preferred api version. If the preferredVersion not exist, the first
// available version will be returned.
func (c *KubeClient) GetApiResourceByName(name, preferredVersion string) (*metav1.APIResource, error) {
	var cans []*metav1.APIResource
	getFunc := func() error {
		resources, err := c.GetApiResourceList()
		if err != nil {
			return errors.Trace(NewTypeNotFoundError(fmt.Sprintf("find apiResource for name error: %s %s", c.cluster, name)))
		}

		for _, rl := range resources {
			for idx, r := range rl.APIResources {
				if r.Name == name {
					cans = append(cans, &rl.APIResources[idx])
				}
			}
		}

		if len(cans) == 0 {
			return errors.Trace(NewTypeNotFoundError(fmt.Sprintf("find apiResource for name error: %s %s", c.cluster, name)))
		}
		return nil
	}

	err := getFunc()
	if err != nil {
		if IsResourceTypeNotFound(err) {
			// Force sync to make sure the resource type is not exist
			if err := c.syncAPIResourceMap(true); err != nil {
				return nil, err
			}
			err = getFunc()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
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
	version, err := c.getVersionByGroup(group)
	if err != nil {
		if IsResourceTypeNotFound(err) {
			if err := c.syncGroupVersion(true); err != nil {
				return "", err
			}
			return c.getVersionByGroup(group)
		}
	}
	return version, err
}

func (c *KubeClient) getVersionByGroup(group string) (string, error) {
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
func (c *KubeClient) GetGroupVersionByName(name, preferredVersion string) (schema.GroupVersion, error) {
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
	klog.V(3).Infof("force resync group version info for cluster: %s", c.cluster)
	groups, err := c.kc.Discovery().ServerGroups()
	if err != nil {
		return err
	}
	klog.V(5).Infof("resync group version info %+v for cluster: %s", groups, c.cluster)
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
	klog.V(3).Infof("force resync api resources for cluster: %s", c.cluster)
	_, serverResourceList, err := c.kc.Discovery().ServerGroupsAndResources()
	if err != nil {
		// ignore GroupDiscoveryFailedError
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return err
		}
		// ensure serverResourceList is a secure var
		if serverResourceList == nil {
			return err
		}
		klog.Warningf("%v", err)
	}
	klog.V(5).Infof("resync api resource info %+v for cluster: %s", serverResourceList, c.cluster)

	// set group and version
	for _, rl := range serverResourceList {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			klog.Errorf("parse group version for %s error: %s", rl.GroupVersion, err)
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
func (c *KubeClient) ConfigForResource(name, preferredVersion string) (rest.Config, error) {
	newCfg := *c.config
	gv, err := c.GetGroupVersionByName(name, preferredVersion)

	klog.V(3).Infof("Found gv %s for %s", gv.String(), name)
	if err != nil {
		return newCfg, err
	}

	newCfg.GroupVersion = &gv

	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	newCfg.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})
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

func (c *KubeClient) IsNamespaceScoped(resource string) (bool, error) {
	res, err := c.GetApiResourceByName(resource, "")
	if err != nil {
		return false, err
	}
	return res.Namespaced, nil
}

// DynamicClientForResource get dynamic client for resource
func (c *KubeClient) DynamicClientForResource(resource, version string) (dynamic.NamespaceableResourceInterface, error) {
	gv, err := c.GetGroupVersionByName(resource, version)
	if err != nil {
		return nil, err
	}
	gvr := gv.WithResource(resource)

	return c.ic.Resource(gvr), nil
}

// DynamicClientForResource get dynamic client for resource
func (c *KubeClient) DynamicClientForGroupKind(gk metav1.GroupKind) (dynamic.NamespaceableResourceInterface, error) {
	version, err := c.GetVersionByGroup(gk.Group)
	if err != nil {
		return nil, err
	}
	resource, err := c.GetResourceTypeByGroupKind(gk)
	if err != nil {
		return nil, err
	}
	gv := schema.GroupVersion{
		Group:   gk.Group,
		Version: version,
	}
	gvr := gv.WithResource(resource)
	return c.ic.Resource(gvr), nil
}

func (c *KubeClient) ClientForGVK(gvk schema.GroupVersionKind) (dynamic.NamespaceableResourceInterface, error) {
	resource, err := c.GetResourceTypeByKind(gvk.Kind)
	if err != nil {
		return nil, err
	}

	version := gvk.GroupVersion().String()
	return c.DynamicClientForResource(resource, version)
}
