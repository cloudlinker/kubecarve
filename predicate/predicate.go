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

var _ Predicate = Funcs{}
var _ Predicate = ResourceVersionChangedPredicate{}

type Funcs struct {
	IgnoreCreateFunc  func(event.CreateEvent) bool
	IgnoreDeleteFunc  func(event.DeleteEvent) bool
	IgnoreUpdateFunc  func(event.UpdateEvent) bool
	IgnoreGenericFunc func(event.GenericEvent) bool
}

func (p Funcs) IgnoreCreate(e event.CreateEvent) bool {
	if p.IgnoreCreateFunc != nil {
		return p.IgnoreCreateFunc(e)
	}
	return false
}

func (p Funcs) IgnoreDelete(e event.DeleteEvent) bool {
	if p.IgnoreDeleteFunc != nil {
		return p.IgnoreDeleteFunc(e)
	}
	return false
}

func (p Funcs) IgnoreUpdate(e event.UpdateEvent) bool {
	if p.IgnoreUpdateFunc != nil {
		return p.IgnoreUpdateFunc(e)
	}
	return false
}

func (p Funcs) IgnoreGeneric(e event.GenericEvent) bool {
	if p.IgnoreGenericFunc != nil {
		return p.IgnoreGenericFunc(e)
	}
	return false
}

type ResourceVersionChangedPredicate struct {
	Funcs
}

func (ResourceVersionChangedPredicate) IgnoreUpdate(e event.UpdateEvent) bool {
	if e.MetaOld == nil ||
		e.ObjectOld == nil ||
		e.ObjectNew == nil ||
		e.MetaNew == nil ||
		e.MetaNew.GetResourceVersion() == e.MetaOld.GetResourceVersion() {
		return true
	}

	return false
}
