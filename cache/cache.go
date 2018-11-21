package cache

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/cloudlinker/kubecarve/cache/internal"
	"github.com/cloudlinker/kubecarve/client"
	"github.com/cloudlinker/kubecarve/client/apiutil"
)

// IndexerFunc knows how to take an object and turn it into a series
// of (non-namespaced) keys for that object.
type IndexerFunc func(runtime.Object) []string

// FieldIndexer knows how to index over a particular "field" such that it
// can later be used by a field selector.
type FieldIndexer interface {
	IndexField(obj runtime.Object, field string, extractValue IndexerFunc) error
}

type Informers interface {
	GetInformer(obj runtime.Object) (toolscache.SharedIndexInformer, error)
	GetInformerForKind(gvk schema.GroupVersionKind) (toolscache.SharedIndexInformer, error)
	Start(stopCh <-chan struct{}) error
	WaitForCacheSync(stop <-chan struct{}) bool
	IndexField(obj runtime.Object, field string, extractValue IndexerFunc) error
}

type Cache interface {
	client.Reader
	Informers
}

type Options struct {
	Scheme    *runtime.Scheme
	Mapper    meta.RESTMapper
	Resync    *time.Duration
	Namespace string
}

var defaultResyncTime = 10 * time.Hour

func New(config *rest.Config, opts Options) (Cache, error) {
	opts, err := defaultOpts(config, opts)
	if err != nil {
		return nil, err
	}
	im := internal.NewInformersMap(config, opts.Scheme, opts.Mapper, *opts.Resync, opts.Namespace)
	return &informerCache{InformersMap: im}, nil
}

func defaultOpts(config *rest.Config, opts Options) (Options, error) {
	if opts.Scheme == nil {
		opts.Scheme = scheme.Scheme
	}

	if opts.Mapper == nil {
		var err error
		opts.Mapper, err = apiutil.NewDiscoveryRESTMapper(config)
		if err != nil {
			return opts, fmt.Errorf("could not create RESTMapper from config")
		}
	}

	if opts.Resync == nil {
		opts.Resync = &defaultResyncTime
	}
	return opts, nil
}
