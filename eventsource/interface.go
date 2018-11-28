package eventsource

import (
	"github.com/cloudlinker/kubecarve/predicate"
)

type EventSource interface {
	GetEventChannel(predicates ...predicate.Predicate) (<-chan interface{}, error)
}
