package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/bwagner5/kube-demo/pkg/model"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/utils/env"
)

var (
	defaultKubeConfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", env.WithDefaultString("KUBECONFIG", defaultKubeConfigPath))
	if err != nil {
		log.Fatalf("could not initialize kubeconfig: %v", err)
	}

	ctx := context.Background()
	cluster := startControllers(ctx, config)
	p := tea.NewProgram(model.NewModel(cluster))
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func startControllers(ctx context.Context, config *rest.Config) *state.Cluster {
	kubeClient, err := client.New(config, client.Options{})
	if err != nil {
		log.Fatalf("could not initialize kube-client: %v", err)
	}

	cluster := state.NewCluster(&clock.RealClock{}, kubeClient)
	manager := state.NewManagerOrDie(ctx, config)
	if err := state.Register(ctx, manager, cluster); err != nil {
		log.Fatalf("%v", err)
	}
	go func() {
		if err := manager.Start(ctx); err != nil {
			log.Fatalf("%v", err)
		}
	}()
	return cluster
}
