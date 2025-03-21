// Copyright (C) 2025 ingress-anubis contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: GPL-3.0

// Package controller contains the ingress controller implementation.
package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/jaredallard/ingress-anubis/internal/config"
	"go.rgst.io/stencil/v2/pkg/slogext"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KubernetesService is the concrete implementation of the serviceActivity interface
// which defines methods to start and stop a service. In this case the service
// being implemented is a kubernetes controller/webhook.
type KubernetesService struct {
	scheme *runtime.Scheme
	log    slogext.Logger
}

// NewKubernetesService creates a new KubernetesService instance
// scoped to this particular scheme.
func NewKubernetesService(log slogext.Logger) *KubernetesService {
	return &KubernetesService{
		log:    log,
		scheme: runtime.NewScheme(),
	}
}

// Run starts a Kubernetes controller/webhook.
//
// Run returns on context cancellation, on a call to Close, or on failure.
func (s *KubernetesService) Run(ctx context.Context) error {
	log.SetLogger(logr.FromSlogHandler(s.log.GetHandler()))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger: logr.FromSlogHandler(s.log.GetHandler()),
		// LeaderElection:          true,
		// LeaderElectionID:        "ingress-anubis.jaredallard.github.io",
		// LeaderElectionNamespace: "ingress-anubis", // TODO(jaredallard): Configurable
	})
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(&IngressReconciler{log: s.log, cfg: &config.Config{Namespace: "ingress-anubis"}, client: mgr.GetClient()})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return mgr.Start(ctx)
}

// Close cleans up webhooks and controllers managed by this instance.
func (s *KubernetesService) Close(_ context.Context) error {
	// TODO(jaredallard): Implement
	return nil
}
