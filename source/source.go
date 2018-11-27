package source

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"

	"github.com/cloudlinker/kubecarve/cache"
	"github.com/cloudlinker/kubecarve/event"
	"github.com/cloudlinker/kubecarve/handler"
	"github.com/cloudlinker/kubecarve/predicate"
	toolscache "k8s.io/client-go/tools/cache"
)

const (
	defaultBufferSize = 1024
)

type Source interface {
	Start(handler.EventHandler, workqueue.RateLimitingInterface, ...predicate.Predicate) error
}

type Kind struct {
	Type  runtime.Object
	cache cache.Cache
}

var _ Source = &Kind{}

func (ks *Kind) Start(handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {

	if ks.Type == nil {
		return fmt.Errorf("must specify Kind.Type")
	}

	if ks.cache == nil {
		return fmt.Errorf("must call CacheInto on Kind before calling Start")
	}

	i, err := ks.cache.GetInformer(ks.Type)
	if err != nil {
		return err
	}
	i.AddEventHandler(newHandlerAdaptor(handler, queue, predicates))
	return nil
}

func (ks *Kind) String() string {
	if ks.Type != nil && ks.Type.GetObjectKind() != nil {
		return fmt.Sprintf("kind source: %v", ks.Type.GetObjectKind().GroupVersionKind().String())
	}
	return fmt.Sprintf("kind source: unknown GVK")
}

var _ Source = &Channel{}

// Channel is used to provide a source of events originating outside the cluster
// (e.g. GitHub Webhook callback).  Channel requires the user to wire the external
// source (eh.g. http handler) to write GenericEvents to the underlying channel.
type Channel struct {
	once           sync.Once
	Source         <-chan event.GenericEvent
	stop           <-chan struct{}
	dest           []chan event.GenericEvent
	DestBufferSize int
	destLock       sync.Mutex
}

func (cs *Channel) String() string {
	return fmt.Sprintf("channel source: %p", cs)
}

func (cs *Channel) Start(
	handler handler.EventHandler,
	queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	if cs.Source == nil {
		return fmt.Errorf("must specify Channel.Source")
	}

	if cs.stop == nil {
		return fmt.Errorf("must call InjectStop on Channel before calling Start")
	}

	if cs.DestBufferSize == 0 {
		cs.DestBufferSize = defaultBufferSize
	}

	cs.once.Do(func() {
		go cs.syncLoop()
	})

	dst := make(chan event.GenericEvent, cs.DestBufferSize)
	go func() {
		for evt := range dst {
			shouldHandle := true
			for _, p := range predicates {
				if p.IgnoreGeneric(evt) {
					shouldHandle = false
					break
				}
			}

			if shouldHandle {
				handler.Generic(evt, queue)
			}
		}
	}()

	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	cs.dest = append(cs.dest, dst)
	return nil
}

func (cs *Channel) doStop() {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.dest {
		close(dst)
	}
}

func (cs *Channel) distribute(evt event.GenericEvent) {
	cs.destLock.Lock()
	defer cs.destLock.Unlock()

	for _, dst := range cs.dest {
		dst <- evt
	}
}

func (cs *Channel) syncLoop() {
	for {
		select {
		case <-cs.stop:
			cs.doStop()
			return
		case evt := <-cs.Source:
			cs.distribute(evt)
		}
	}
}

type Informer struct {
	Informer toolscache.SharedIndexInformer
}

var _ Source = &Informer{}

func (is *Informer) Start(handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {

	if is.Informer == nil {
		return fmt.Errorf("must specify Informer.Informer")
	}

	is.Informer.AddEventHandler(newHandlerAdaptor(handler, queue, predicates))
	return nil
}

func (is *Informer) String() string {
	return fmt.Sprintf("informer source: %p", is.Informer)
}

type Func func(handler.EventHandler, workqueue.RateLimitingInterface, ...predicate.Predicate) error

func (f Func) Start(evt handler.EventHandler, queue workqueue.RateLimitingInterface,
	pr ...predicate.Predicate) error {
	return f(evt, queue, pr...)
}

func (f Func) String() string {
	return fmt.Sprintf("func source: %p", f)
}
