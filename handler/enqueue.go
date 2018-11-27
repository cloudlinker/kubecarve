package handler

import (
	"k8s.io/client-go/util/workqueue"

	"github.com/cloudlinker/kubecarve/event"
)

var _ EventHandler = &EnqueueEvent{}

type EnqueueEvent struct{}

func (h *EnqueueEvent) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	if e.Meta == nil {
		return
	}

	q.Add(e)
}

func (h *EnqueueEvent) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if e.MetaOld == nil || e.MetaNew == nil {
		return
	}

	q.Add(e)
}

func (h *EnqueueEvent) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if e.Meta == nil {
		return
	}

	q.Add(e)
}

func (h *EnqueueEvent) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	if e.Meta == nil {
		return
	}

	q.Add(e)
}
