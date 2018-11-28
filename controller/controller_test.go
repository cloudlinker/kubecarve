package controller

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	ut "github.com/cloudlinker/cement/unittest"
	"github.com/cloudlinker/kubecarve/cache"
	"github.com/cloudlinker/kubecarve/client"
	"github.com/cloudlinker/kubecarve/event"
	"github.com/cloudlinker/kubecarve/handler"
	"github.com/cloudlinker/kubecarve/testenv"
)

type dumbEventHandler struct {
	podCreateEvent      int
	podUpdateEventCount int
	podDeleteEventCount int
}

func (d *dumbEventHandler) OnCreate(e event.CreateEvent) (handler.Result, error) {
	if _, ok := e.Object.(*corev1.Pod); ok {
		d.podCreateEvent += 1
	}
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	if _, ok := e.ObjectOld.(*corev1.Pod); ok {
		d.podUpdateEventCount += 1
	}
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	if _, ok := e.Object.(*corev1.Pod); ok {
		d.podDeleteEventCount += 1
	}
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func newPod(name, ns string, labels map[string]string, restartPolicy corev1.RestartPolicy) *corev1.Pod {
	three := int64(3)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers:            []corev1.Container{{Name: "nginx", Image: "nginx"}},
			RestartPolicy:         restartPolicy,
			ActiveDeadlineSeconds: &three,
		},
	}
}

func TestController(t *testing.T) {
	env := testenv.NewEnv(os.Getenv("K8S_ASSETS"), nil)
	err := env.Start()
	ut.Assert(t, err == nil, "testenv cluster start failed:%v", err)
	defer func() {
		env.Stop()
	}()

	cli, err := client.New(env.Config, client.Options{})
	ut.Assert(t, err == nil, "create client failed:%v", err)

	stop := make(chan struct{})
	defer close(stop)
	c, err := cache.New(env.Config, cache.Options{})
	ut.Assert(t, err == nil, "create cache failed:%v", err)
	go c.Start(stop)
	ut.Assert(t, c.WaitForCacheSync(stop), "wait for sync should ok")

	ctrl := New("dumbController", c, scheme.Scheme)
	ctrl.Watch(&corev1.Pod{})
	handler := &dumbEventHandler{}
	go ctrl.Start(handler, stop)

	testNamespaceOne := "test-namespace-1"
	testNamespaceTwo := "test-namespace-2"
	err = cli.Create(context.TODO(), newPod("test-pod-1", testNamespaceOne, map[string]string{"test-label": "test-pod-1"}, corev1.RestartPolicyNever))
	ut.Assert(t, err == nil, "create pod failed:%v", err)
	err = cli.Create(context.TODO(), newPod("test-pod-2", testNamespaceTwo, map[string]string{"test-label": "test-pod-2"}, corev1.RestartPolicyAlways))
	ut.Assert(t, err == nil, "create pod failed:%v", err)
	err = cli.Create(context.TODO(), newPod("test-pod-3", testNamespaceTwo, map[string]string{"test-label": "test-pod-3"}, corev1.RestartPolicyOnFailure))
	ut.Assert(t, err == nil, "create pod failed:%v", err)

	<-time.After(time.Second)
	ut.Equal(t, handler.podCreateEvent, 3)
	ut.Equal(t, handler.podUpdateEventCount, 0)
	ut.Equal(t, handler.podDeleteEventCount, 0)
}
