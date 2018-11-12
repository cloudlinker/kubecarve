package internal

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type InformersMap struct {
	structured *specificInformersMap
	Scheme     *runtime.Scheme
}

func NewInformersMap(config *rest.Config,
	scheme *runtime.Scheme,
	mapper meta.RESTMapper,
	resync time.Duration,
	namespace string) *InformersMap {

	return &InformersMap{
		structured: newStructuredInformersMap(config, scheme, mapper, resync, namespace),
		Scheme:     scheme,
	}
}

func (m *InformersMap) Start(stop <-chan struct{}) error {
	go m.structured.Start(stop)
	<-stop
	return nil
}

func (m *InformersMap) WaitForCacheSync(stop <-chan struct{}) bool {
	syncedFuncs := append([]cache.InformerSynced(nil), m.structured.HasSyncedFuncs()...)
	return cache.WaitForCacheSync(stop, syncedFuncs...)
}

func (m *InformersMap) Get(gvk schema.GroupVersionKind, obj runtime.Object) (*MapEntry, error) {
	return m.structured.Get(gvk, obj)
}

func newStructuredInformersMap(config *rest.Config, scheme *runtime.Scheme, mapper meta.RESTMapper, resync time.Duration, namespace string) *specificInformersMap {
	return newSpecificInformersMap(config, scheme, mapper, resync, namespace, createStructuredListWatch)
}
