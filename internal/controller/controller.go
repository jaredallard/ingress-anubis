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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KubernetesService contains all of the setup and logic for the
// Kubernetes controller(s).
type KubernetesService struct {
	log slogext.Logger
	cfg *config.Config
}

// NewKubernetesService creates a new [KubernetesService] instance.
func NewKubernetesService(cfg *config.Config, log slogext.Logger) *KubernetesService {
	return &KubernetesService{log, cfg}
}

// Run starts the kubernetes controller(s)
func (s *KubernetesService) Run(ctx context.Context) error {
	log.SetLogger(logr.FromSlogHandler(s.log.GetHandler()))

	opts := ctrl.Options{
		Logger: logr.FromSlogHandler(s.log.GetHandler()),
	}
	if s.cfg.LeaderElection {
		opts.LeaderElection = true
		opts.LeaderElectionID = "ingress-anubis.jaredallard.github.io"
		opts.LeaderElectionNamespace = s.cfg.Namespace
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	if err := builder.
		ControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(&IngressReconciler{s.log, s.cfg, mgr.GetClient()}); err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return mgr.Start(ctx)
}
