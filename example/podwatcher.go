package main

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/cloudlinker/kubecarve/cache"
	"github.com/cloudlinker/kubecarve/client/config"
	"github.com/cloudlinker/kubecarve/controller"
	"github.com/cloudlinker/kubecarve/event"
	"github.com/cloudlinker/kubecarve/handler"
)

type dumbEventHandler struct {
	podCreateEvent      int
	podUpdateEventCount int
	podDeleteEventCount int
}

func (d *dumbEventHandler) OnCreate(e event.CreateEvent) (handler.Result, error) {
	fmt.Printf("---> create kind [%v] with name [%s]\n", e.Object.GetObjectKind(), e.Meta.GetName())
	d.podCreateEvent += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	fmt.Printf("---> update kind [%v] with name [%s]\n", e.ObjectOld.GetObjectKind(), e.MetaOld.GetName())
	d.podUpdateEventCount += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	fmt.Printf("---> delete kind [%v] with name [%s]\n", e.Object.GetObjectKind(), e.Meta.GetName())
	d.podDeleteEventCount += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Printf("---> get config failed %v\n", err)
		return
	}

	stop := make(chan struct{})
	defer close(stop)

	c, err := cache.New(cfg, cache.Options{})
	if err != nil {
		fmt.Printf("---> create cache failed %v\n", err)
		return
	}
	go c.Start(stop)

	c.WaitForCacheSync(stop)
	fmt.Printf("---> cache synced\n")

	ctrl := controller.New("dumbController", c, scheme.Scheme)
	ctrl.Watch(&corev1.Pod{})
	handler := &dumbEventHandler{}
	ctrl.Start(handler, stop)
}
