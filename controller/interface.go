package controller

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cloudlinker/kubecarve/handler"
	"github.com/cloudlinker/kubecarve/predicate"
)

type Controller interface {
	Watch(obj runtime.Object, predicates ...predicate.Predicate) error
	Start(handler handler.EventHandler, stop <-chan struct{})
}
