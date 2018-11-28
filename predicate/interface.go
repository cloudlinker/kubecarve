package predicate

import (
	"github.com/cloudlinker/kubecarve/event"
)

type Predicate interface {
	IgnoreCreate(event.CreateEvent) bool
	IgnoreDelete(event.DeleteEvent) bool
	IgnoreUpdate(event.UpdateEvent) bool
	IgnoreGeneric(event.GenericEvent) bool
}
