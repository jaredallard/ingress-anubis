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

package config

import (
	"fmt"
	"strconv"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"
)

// AnnotationKey is an annotation supported by [IngressConfig].
type AnnotationKey string

// String implements the stringer interface.
func (ak AnnotationKey) String() string {
	return string(ak)
}

// Contains valid annotations used by [IngressConfig].
const (
	// AnnotationKeyBase is the base of annotations supported.
	AnnotationKeyBase = "ingress-anubis.jaredallard.github.com/"

	// AnnotationKeyDifficulty is used by [IngressConfig.Difficulty].
	AnnotationKeyDifficulty AnnotationKey = AnnotationKeyBase + "difficulty"

	// AnnotationKeyServeRobotsTxt is used by
	// [IngressConfig.ServeRobotsTxt].
	AnnotationKeyServeRobotsTxt AnnotationKey = AnnotationKeyBase + "serve-robots-txt"

	// AnnotationKeyIngressClass is used by [IngressConfig.IngressClass].
	AnnotationKeyIngressClass AnnotationKey = AnnotationKeyBase + "ingress-class"
)

// AnnotationKeys contains all valid [AnnotationKey] values.
var AnnotationKeys = [...]AnnotationKey{
	AnnotationKeyDifficulty,
	AnnotationKeyServeRobotsTxt,
	AnnotationKeyIngressClass,
}

// IngressConfig contains configuration from an ingress object.
type IngressConfig struct {
	// Difficulty is the difficulty parameter to pass to anubis.
	// See: https://anubis.techaro.lol/docs/admin/installation
	Difficulty *int

	// ServeRobotsTxt enables serving robots.txt. Enabled by default.
	// See: https://anubis.techaro.lol/docs/admin/installation
	ServeRobotsTxt *bool

	// IngressClass denotes which ingress class should be used by the
	// controller instead of the default. The default comes from
	// [Config.WrappedIngressClassName].
	IngressClass *string
}

// applyDefaults applies defaults to the provided [IngressConfig].
func applyDefaults(ic *IngressConfig) {
	if ic.Difficulty == nil {
		ic.Difficulty = ptr.To(4)
	}

	if ic.ServeRobotsTxt == nil {
		ic.ServeRobotsTxt = ptr.To(true)
	}
}

// GetIngressConfigFromIngress returns an [IngressConfig] from the
// provided [networkingv1.Ingress]. If no options are found, the default
// configuration is returned. An error is only returned if the provided
// ingress contains invalid configuration data (e.g., int expected, but
// got non-int)
func GetIngressConfigFromIngress(ing *networkingv1.Ingress) (*IngressConfig, error) {
	cfg := IngressConfig{}

	// Capture values from the annotations, if present.
	if ing != nil && ing.Annotations != nil {
		for _, k := range AnnotationKeys {
			v, ok := ing.Annotations[string(k)]
			if !ok {
				continue
			}

			switch k {
			case AnnotationKeyServeRobotsTxt:
				b, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("failed to parse annotation %s value %q as bool", AnnotationKeyServeRobotsTxt, v)
				}
				cfg.ServeRobotsTxt = &b
			case AnnotationKeyDifficulty:
				d, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("failed to parse annotation %s value %q as int", AnnotationKeyDifficulty, v)
				}
				cfg.Difficulty = &d
			case AnnotationKeyIngressClass:
				cfg.IngressClass = &v
			default:
				panic(fmt.Errorf("unknown annotation key %q", string(k)))
			}
		}
	}

	applyDefaults(&cfg)

	return &cfg, nil
}
