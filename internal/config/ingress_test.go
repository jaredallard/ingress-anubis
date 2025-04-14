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
	"testing"

	"github.com/google/go-cmp/cmp"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func getDefaults() *IngressConfig {
	var cfg IngressConfig
	applyDefaults(&cfg)
	return &cfg
}

func TestGetIngressConfigFromIngress(t *testing.T) {
	type args struct {
		ing *networkingv1.Ingress
	}

	ing := func(a map[AnnotationKey]string) *networkingv1.Ingress {
		annotations := make(map[string]string)
		for k, v := range a {
			annotations[string(k)] = v
		}
		return &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}}
	}

	defplus := func(overrides IngressConfig) *IngressConfig {
		resp := getDefaults()
		if overrides.Difficulty != nil {
			resp.Difficulty = overrides.Difficulty
		}
		if overrides.ServeRobotsTxt != nil {
			resp.ServeRobotsTxt = overrides.ServeRobotsTxt
		}
		if overrides.IngressClass != nil {
			resp.IngressClass = overrides.IngressClass
		}
		if overrides.OGPassthrough != nil {
			resp.OGPassthrough = overrides.OGPassthrough
		}
		return resp
	}

	tests := []struct {
		name    string
		args    args
		want    *IngressConfig
		wantErr bool
	}{
		{
			name: "should support no annotations",
			args: args{},
			want: getDefaults(),
		},
		{
			name: "should support setting difficulty",
			args: args{ing(map[AnnotationKey]string{
				AnnotationKeyDifficulty: "5",
			})},
			want: defplus(IngressConfig{Difficulty: ptr.To(5)}),
		},
		{
			name: "should support setting ServeRobotsTxt",
			args: args{ing(map[AnnotationKey]string{
				AnnotationKeyServeRobotsTxt: "false",
			})},
			want: defplus(IngressConfig{ServeRobotsTxt: ptr.To(false)}),
		},
		{
			name: "should support setting IngressClass",
			args: args{ing(map[AnnotationKey]string{
				AnnotationKeyIngressClass: "traefik",
			})},
			want: defplus(IngressConfig{IngressClass: ptr.To("traefik")}),
		},
		{
			name: "should support setting OGPassthrough",
			args: args{ing(map[AnnotationKey]string{
				AnnotationKeyOGPassthrough: "false",
			})},
			want: defplus(IngressConfig{OGPassthrough: ptr.To(false)}),
		},
		{
			name: "should fail when invalid value is set for key",
			args: args{ing(map[AnnotationKey]string{
				AnnotationKeyServeRobotsTxt: "bfalse",
			})},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIngressConfigFromIngress(tt.args.ing)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIngressConfigFromIngress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetIngressConfigFromIngress() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
