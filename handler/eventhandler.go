package handler

import (
	"k8s.io/client-go/util/workqueue"

	"github.com/cloudlinker/kubecarve/event"
)

type EventHandler interface {
	Create(event.CreateEvent, workqueue.RateLimitingInterface)
	Update(event.UpdateEvent, workqueue.RateLimitingInterface)
	Delete(event.DeleteEvent, workqueue.RateLimitingInterface)
	Generic(event.GenericEvent, workqueue.RateLimitingInterface)
}
