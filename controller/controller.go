package controller

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	"github.com/cloudlinker/kubecarve/cache"
	"github.com/cloudlinker/kubecarve/client/apiutil"
	"github.com/cloudlinker/kubecarve/event"
	"github.com/cloudlinker/kubecarve/eventsource"
	"github.com/cloudlinker/kubecarve/handler"
	"github.com/cloudlinker/kubecarve/predicate"
)

type controller struct {
	name    string
	handler handler.EventHandler
	cache   cache.Cache
	sources map[schema.GroupVersionKind]<-chan interface{}
	queue   workqueue.RateLimitingInterface
	scheme  *runtime.Scheme
}

func New(name string, cache cache.Cache, scheme *runtime.Scheme) Controller {
	return &controller{
		name:    name,
		cache:   cache,
		sources: make(map[schema.GroupVersionKind]<-chan interface{}),
		scheme:  scheme,
	}
}

func (c *controller) Watch(obj runtime.Object, predicates ...predicate.Predicate) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	if _, ok := c.sources[gvk]; ok {
		return fmt.Errorf("watch obj %v more than once", gvk)
	}

	ch, err := eventsource.New(gvk, c.cache).GetEventChannel(predicates...)
	if err != nil {
		return err
	}

	c.sources[gvk] = ch
	return nil
}

func (c *controller) Start(handler handler.EventHandler, stop <-chan struct{}) {
	c.handler = handler
	c.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), c.name)

	var wg wait.Group
	wg.StartWithChannel(stop, c.collectEvent)
	wg.StartWithChannel(stop, c.processEvent)
	wg.Wait()
}

func (c *controller) collectEvent(stop <-chan struct{}) {
	cases := make([]reflect.SelectCase, 0, len(c.sources)+1)
	for _, ch := range c.sources {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(stop),
	})

	for len(cases) > 0 {
		i, e, ok := reflect.Select(cases)
		if i == len(cases)-1 {
			c.queue.ShutDown()
			return
		}

		if !ok {
			cases = append(cases[:i], cases[i+1:]...)
			continue
		}

		c.queue.Add(e.Interface())
	}
}

func (c *controller) processEvent(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		c.processNextEvent()
	}
}

func (c *controller) processNextEvent() {
	o, shutdown := c.queue.Get()
	if shutdown {
		return
	}
	defer c.queue.Done(o)

	if o == nil {
		c.queue.Forget(o)
		return
	}

	var err error
	var result handler.Result
	switch e := o.(type) {
	case event.CreateEvent:
		result, err = c.handler.OnCreate(e)
	case event.UpdateEvent:
		result, err = c.handler.OnUpdate(e)
	case event.DeleteEvent:
		result, err = c.handler.OnDelete(e)
	case event.GenericEvent:
		result, err = c.handler.OnGeneric(e)
	default:
		panic(fmt.Sprintf("unkown event [%v]", reflect.TypeOf(o).Name()))
	}

	if err != nil {
		c.queue.AddRateLimited(o)
	} else if result.RequeueAfter > 0 {
		c.queue.AddAfter(o, result.RequeueAfter)
	} else if result.Requeue {
		c.queue.AddRateLimited(o)
	} else {
		c.queue.Forget(o)
	}
}
