package eventsource

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cloudlinker/kubecarve/cache"
	"github.com/cloudlinker/kubecarve/predicate"
)

const (
	defaultBufferSize = 1024
)

type resourceEventSource struct {
	typ   runtime.Object
	cache cache.Cache
}

var _ EventSource = &resourceEventSource{}

func New(typ runtime.Object, cache cache.Cache) EventSource {
	return &resourceEventSource{
		typ:   typ,
		cache: cache,
	}
}

func (l *resourceEventSource) GetEventChannel(predicates ...predicate.Predicate) (<-chan interface{}, error) {
	i, err := l.cache.GetInformer(l.typ)
	if err != nil {
		return nil, err
	}

	ch := make(chan interface{})
	i.AddEventHandler(newHandlerAdaptor(predicates, ch))
	return ch, nil
}

func (l *resourceEventSource) String() string {
	if l.typ != nil && l.typ.GetObjectKind() != nil {
		return fmt.Sprintf("kind source: %v", l.typ.GetObjectKind().GroupVersionKind().String())
	}
	return fmt.Sprintf("kind source: unknown GVK")
}
