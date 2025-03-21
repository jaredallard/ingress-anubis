package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jaredallard/ingress-anubis/internal/controller"
	"go.rgst.io/stencil/v2/pkg/slogext"
)

func entrypoint(log slogext.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	svc := controller.NewKubernetesService(log)
	defer svc.Close(context.Background())

	if err := svc.Run(ctx); err != nil {
		return err
	}

	return nil
}

func main() {
	log := slogext.New()
	if err := entrypoint(log); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", err)
		os.Exit(1)
	}
}
