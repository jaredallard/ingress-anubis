// Package main implements the entrypoint of the ingress-anubis
// controller.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jaredallard/ingress-anubis/internal/config"
	"github.com/jaredallard/ingress-anubis/internal/controller"
	"go.rgst.io/stencil/v2/pkg/slogext"
)

func entrypoint(log slogext.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	svc := controller.NewKubernetesService(cfg, log)
	return svc.Run(ctx)
}

func main() {
	log := slogext.New()
	if err := entrypoint(log); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", err)
		os.Exit(1)
	}
}
