package controller

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cloudlinker/kubecarve/handler"
	"github.com/cloudlinker/kubecarve/predicate"
)

type Controller interface {
	Watch(obj runtime.Object) error
	Start(stop <-chan struct{}, handler handler.EventHandler, predicates ...predicate.Predicate)
}
