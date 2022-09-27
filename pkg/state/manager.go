package state

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func Register(ctx context.Context, manager manager.Manager, cluster *Cluster) error {
	return multierr.Combine(
		NewNodeController(manager.GetClient(), cluster).Register(ctx, manager),
		NewPodController(manager.GetClient(), cluster).Register(ctx, manager),
	)
}

// NewManagerOrDie instantiates a controller manager or panics
func NewManagerOrDie(ctx context.Context, config *rest.Config, options controllerruntime.Options) manager.Manager {
	newManager, err := controllerruntime.NewManager(config, options)
	if err != nil {
		panic(fmt.Sprintf("Failed to create controller newManager, %s", err))
	}
	if err := newManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, "spec.nodeName", func(o client.Object) []string {
		return []string{o.(*corev1.Pod).Spec.NodeName}
	}); err != nil {
		panic(fmt.Sprintf("Failed to setup pod indexer, %s", err))
	}
	return newManager
}
