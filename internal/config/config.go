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

// Package config contains the configuration.
package config

import "github.com/caarlos0/env/v11"

// Config contains the configuration
type Config struct {
	// Namespace that the ingress controller is running in and should
	// create resources in.
	Namespace string `env:"NAMESPACE" envDefault:"ingress-anubis"`

	// AnubisVersion is the version of Anubis to use. If not set, then the
	// latest version known to the controller at build time will be used.
	//renovate: datasource=github-tags depName=anubis packageName=techarohq/anubis
	AnubisVersion string `env:"ANUBIS_VERSION" envDefault:"v1.17.0"`

	// AnubisImage is the docker image to use, note that the version (tag)
	// comes from [Config.AnubisVersion].
	AnubisImage string `env:"ANUBIS_IMAGE" envDefault:"ghcr.io/techarohq/anubis"`

	// IngressClassName is the ingress class name that Anubis itself
	// should use.
	IngressClassName string `env:"INGRESS_CLASS_NAME" envDefault:"anubis"`

	// WrappedIngressClassName is the name of the ingressClass to use for
	// the ingress managed by anubis. While this is configurable, only
	// nginx has been tested (though, in theory, any should work).
	WrappedIngressClassName string `env:"WRAPPED_INGRESS_CLASS_NAME" envDefault:"nginx"`

	// LeaderElection enables or disables leader election. This should
	// usually always be on.
	LeaderElection bool `env:"LEADER_ELECTION" envDefault:"true"`

	// Annotations is a map of annotations to set on the managed Anubis
	// pod. Example:
	//
	// ANNOTATIONS="prometheus.io/scrape:true,hello.world/a-thing:1"
	Annotations map[string]string `env:"ANNOTATIONS"`
}

// Load returns a configuration object from the environment.
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
