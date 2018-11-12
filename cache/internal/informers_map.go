package internal

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/cloudlinker/kubecarve/client/apiutil"
)

type createListWatcherFunc func(gvk schema.GroupVersionKind, ip *specificInformersMap) (*cache.ListWatch, error)

func newSpecificInformersMap(config *rest.Config,
	scheme *runtime.Scheme,
	mapper meta.RESTMapper,
	resync time.Duration,
	namespace string,
	createListWatcher createListWatcherFunc) *specificInformersMap {
	ip := &specificInformersMap{
		config:            config,
		Scheme:            scheme,
		mapper:            mapper,
		informersByGVK:    make(map[schema.GroupVersionKind]*MapEntry),
		codecs:            serializer.NewCodecFactory(scheme),
		paramCodec:        runtime.NewParameterCodec(scheme),
		resync:            resync,
		createListWatcher: createListWatcher,
		namespace:         namespace,
	}
	return ip
}

type MapEntry struct {
	Informer cache.SharedIndexInformer
	Reader   CacheReader
}

type specificInformersMap struct {
	Scheme            *runtime.Scheme
	config            *rest.Config
	mapper            meta.RESTMapper
	informersByGVK    map[schema.GroupVersionKind]*MapEntry
	codecs            serializer.CodecFactory
	paramCodec        runtime.ParameterCodec
	stop              <-chan struct{}
	resync            time.Duration
	mu                sync.RWMutex
	started           bool
	createListWatcher createListWatcherFunc
	namespace         string
}

func (ip *specificInformersMap) Start(stop <-chan struct{}) {
	ip.mu.Lock()
	ip.stop = stop
	for _, informer := range ip.informersByGVK {
		go informer.Informer.Run(stop)
	}
	ip.started = true
	ip.mu.Unlock()

	<-stop
}

func (ip *specificInformersMap) HasSyncedFuncs() []cache.InformerSynced {
	ip.mu.RLock()
	defer ip.mu.RUnlock()
	syncedFuncs := make([]cache.InformerSynced, 0, len(ip.informersByGVK))
	for _, informer := range ip.informersByGVK {
		syncedFuncs = append(syncedFuncs, informer.Informer.HasSynced)
	}
	return syncedFuncs
}

func (ip *specificInformersMap) Get(gvk schema.GroupVersionKind, obj runtime.Object) (*MapEntry, error) {
	i, ok := func() (*MapEntry, bool) {
		ip.mu.RLock()
		defer ip.mu.RUnlock()
		i, ok := ip.informersByGVK[gvk]
		return i, ok
	}()
	if ok {
		return i, nil
	}

	var sync bool
	i, err := func() (*MapEntry, error) {
		ip.mu.Lock()
		defer ip.mu.Unlock()

		var ok bool
		i, ok := ip.informersByGVK[gvk]
		if ok {
			return i, nil
		}

		var lw *cache.ListWatch
		lw, err := ip.createListWatcher(gvk, ip)
		if err != nil {
			return nil, err
		}
		ni := cache.NewSharedIndexInformer(lw, obj, ip.resync, cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		})
		i = &MapEntry{
			Informer: ni,
			Reader:   CacheReader{indexer: ni.GetIndexer(), groupVersionKind: gvk},
		}
		ip.informersByGVK[gvk] = i

		if ip.started {
			sync = true
			go i.Informer.Run(ip.stop)
		}
		return i, nil
	}()
	if err != nil {
		return nil, err
	}

	if sync {
		if !cache.WaitForCacheSync(ip.stop, i.Informer.HasSynced) {
			return nil, fmt.Errorf("failed waiting for %T Informer to sync", obj)
		}
	}

	return i, err
}

func createStructuredListWatch(gvk schema.GroupVersionKind, ip *specificInformersMap) (*cache.ListWatch, error) {
	mapping, err := ip.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	client, err := apiutil.RESTClientForGVK(gvk, ip.config, ip.codecs)
	if err != nil {
		return nil, err
	}
	listGVK := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	listObj, err := ip.Scheme.New(listGVK)
	if err != nil {
		return nil, err
	}

	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			res := listObj.DeepCopyObject()
			isNamespaceScoped := ip.namespace != "" && mapping.Scope.Name() != meta.RESTScopeNameRoot
			err := client.Get().NamespaceIfScoped(ip.namespace, isNamespaceScoped).Resource(mapping.Resource.Resource).VersionedParams(&opts, ip.paramCodec).Do().Into(res)
			return res, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.Watch = true
			isNamespaceScoped := ip.namespace != "" && mapping.Scope.Name() != meta.RESTScopeNameRoot
			return client.Get().NamespaceIfScoped(ip.namespace, isNamespaceScoped).Resource(mapping.Resource.Resource).VersionedParams(&opts, ip.paramCodec).Watch()
		},
	}, nil
}
