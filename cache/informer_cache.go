package cache

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/cloudlinker/kubecarve/cache/internal"
	"github.com/cloudlinker/kubecarve/client"
	"github.com/cloudlinker/kubecarve/client/apiutil"
)

var (
	_ Informers     = &informerCache{}
	_ client.Reader = &informerCache{}
	_ Cache         = &informerCache{}
)

type informerCache struct {
	*internal.InformersMap
}

func (ip *informerCache) Get(ctx context.Context, key client.ObjectKey, out runtime.Object) error {
	gvk, err := apiutil.GVKForObject(out, ip.Scheme)
	if err != nil {
		return err
	}

	cache, err := ip.InformersMap.Get(gvk, out)
	if err != nil {
		return err
	}
	return cache.Reader.Get(ctx, key, out)
}

func (ip *informerCache) List(ctx context.Context, opts *client.ListOptions, out runtime.Object) error {
	gvk, err := apiutil.GVKForObject(out, ip.Scheme)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("non-list type %T (kind %q) passed as output", out, gvk)
	}

	gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	var cacheTypeObj runtime.Object

	itemsPtr, err := apimeta.GetItemsPtr(out)
	if err != nil {
		return nil
	}

	elemType := reflect.Indirect(reflect.ValueOf(itemsPtr)).Type().Elem()
	cacheTypeValue := reflect.Zero(reflect.PtrTo(elemType))
	var ok bool
	cacheTypeObj, ok = cacheTypeValue.Interface().(runtime.Object)
	if !ok {
		return fmt.Errorf("cannot get cache for %T, its element %T is not a runtime.Object", out, cacheTypeValue.Interface())
	}

	cache, err := ip.InformersMap.Get(gvk, cacheTypeObj)
	if err != nil {
		return err
	}

	return cache.Reader.List(ctx, opts, out)
}

func (ip *informerCache) GetInformerForKind(gvk schema.GroupVersionKind) (cache.SharedIndexInformer, error) {
	obj, err := ip.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	i, err := ip.InformersMap.Get(gvk, obj)
	if err != nil {
		return nil, err
	}
	return i.Informer, err
}

func (ip *informerCache) GetInformer(obj runtime.Object) (cache.SharedIndexInformer, error) {
	gvk, err := apiutil.GVKForObject(obj, ip.Scheme)
	if err != nil {
		return nil, err
	}
	i, err := ip.InformersMap.Get(gvk, obj)
	if err != nil {
		return nil, err
	}
	return i.Informer, err
}

func (ip *informerCache) IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	informer, err := ip.GetInformer(obj)
	if err != nil {
		return err
	}
	return indexByField(informer.GetIndexer(), field, extractValue)
}

func indexByField(indexer cache.Indexer, field string, extractor client.IndexerFunc) error {
	indexFunc := func(objRaw interface{}) ([]string, error) {
		obj, isObj := objRaw.(runtime.Object)
		if !isObj {
			return nil, fmt.Errorf("object of type %T is not an Object", objRaw)
		}
		meta, err := apimeta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		ns := meta.GetNamespace()

		rawVals := extractor(obj)
		var vals []string
		if ns == "" {
			vals = rawVals
		} else {
			vals = make([]string, len(rawVals)*2)
		}
		for i, rawVal := range rawVals {
			vals[i] = internal.KeyToNamespacedKey(ns, rawVal)
			if ns != "" {
				vals[i+len(rawVals)] = internal.KeyToNamespacedKey("", rawVal)
			}
		}

		return vals, nil
	}

	return indexer.AddIndexers(cache.Indexers{internal.FieldIndexName(field): indexFunc})
}
