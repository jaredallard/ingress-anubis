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

package controller

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jaredallard/ingress-anubis/internal/config"
	"go.rgst.io/stencil/v2/pkg/slogext"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Label keys
const (
	// ManagedLabel is the label used to track what is managed by this
	// controller.
	ManagedLabel = "ingress-anubis.jaredallard.github.com/managed"

	// OwningLabel is the label used to store the owning ingress.
	OwningLabel = "ingress-anubis.jaredallard.github.com/owner"
)

// IngressReconciler is the main reconciler of the controller. See
// [IngressReconciler.Reconcile] for more information.
type IngressReconciler struct {
	log    slogext.Logger
	cfg    *config.Config
	client crclient.Client
}

// Reconcile contains the main logic for reconciling all of the
// resources that make up the ingress controller. The following logic is
// documented below:
//
// 1. ingressClassName == anubis
// 2. reconcile deployment
// 3. reconcile service
// 4. reconcile ingress (wrapper/child)
func (ir *IngressReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	origIng := &networkingv1.Ingress{}
	if err := ir.client.Get(ctx, req.NamespacedName, origIng); err != nil {
		return reconcile.Result{}, crclient.IgnoreNotFound(err)
	}

	// Not controlled by us.
	if origIng.Spec.IngressClassName == nil || *origIng.Spec.IngressClassName != "anubis" {
		return reconcile.Result{}, nil
	}

	if origIng.GetLabels()[ManagedLabel] == "true" {
		return reconcile.Result{}, reconcile.TerminalError(fmt.Errorf("attempted to reconcile ingress owned by self"))
	}

	ir.log.Info("reconciling ingress", slog.String("name", req.Name), slog.String("namespace", req.Namespace))

	// Grab the first valid backend from the ingress, we'll use that as
	// anubis' target. Note that technically ingresses can have more than
	// one target, so this won't work in that case.
	var svcBackend *networkingv1.IngressServiceBackend
	if origIng.Spec.DefaultBackend != nil { // Preference to default backend
		svcBackend = origIng.Spec.DefaultBackend.Service
	} else {
		if len(origIng.Spec.Rules) == 0 {
			return reconcile.Result{}, reconcile.TerminalError(fmt.Errorf("no rules or default backend in ingress"))
		}

		rule := origIng.Spec.Rules[0]
		if rule.HTTP == nil {
			return reconcile.Result{}, reconcile.TerminalError(fmt.Errorf("ingress rule 0 HTTP was nil"))
		}

		if len(rule.HTTP.Paths) == 0 {
			return reconcile.Result{}, reconcile.TerminalError(fmt.Errorf("ingress rule 0 paths was empty"))
		}

		path := rule.HTTP.Paths[0]
		svcBackend = path.Backend.Service
	}

	target, err := ir.getTargetFromService(ctx, origIng.Namespace, svcBackend)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileDeployment(ctx, target, req); err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileService(ctx, req); err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileChildIngress(ctx, origIng, req); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// getTargetFromService returns a that can be used to communicate with
// the given service in isb from inside of Kubernetes.
func (ir *IngressReconciler) getTargetFromService(ctx context.Context, ns string, isb *networkingv1.IngressServiceBackend) (string, error) {
	// If the target is a name, we need to look up the service's real
	// port.
	port := isb.Port.Number
	if portName := isb.Port.Name; portName != "" {
		svcKey := crclient.ObjectKey{Namespace: ns, Name: isb.Name}
		var svc corev1.Service
		if err := ir.client.Get(ctx, svcKey, &svc); err != nil {
			return "", fmt.Errorf("failed to look up service for port name translation: %w", err)
		}

		// Find the port
		for _, p := range svc.Spec.Ports {
			if p.Name != portName {
				continue
			}

			port = p.Port
			break
		}
		if port == 0 { // Didn't find it?
			return "", fmt.Errorf("failed to find port %s in service %s", portName, svcKey)
		}
	}

	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", isb.Name, ns, port), nil
}

// reconcileDeployment ensures that a deployment of anubis exists
func (ir *IngressReconciler) reconcileDeployment(ctx context.Context, target string, req reconcile.Request) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ia-" + req.Name,
			Namespace: ir.cfg.Namespace,
		},
	}

	labels := map[string]string{
		"app.kubernetes.io/instance": "anubis",
		"app.kubernetes.io/name":     "anubis",
		ManagedLabel:                 "true",
		OwningLabel:                  req.Namespace + "--" + req.Name,
	}

	_, err := controllerutil.CreateOrUpdate(ctx, ir.client, dep, func() error {
		// Deployment selector is immutable so we set this value only if
		// a new object is going to be created
		if dep.ObjectMeta.CreationTimestamp.IsZero() {
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}

		dep.ObjectMeta.Labels = labels

		// Only one replica is supported by anubis currently
		dep.Spec.Replicas = ptr.To(int32(1))
		dep.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}

		dep.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "main",
					Image: ir.cfg.AnubisImage + ":" + ir.cfg.AnubisVersion,
					Env: []corev1.EnvVar{
						{Name: "BIND", Value: ":8080"},
						{Name: "DIFFICULTY", Value: "4"},
						{Name: "METRICS_BIND", Value: ":9090"},
						{Name: "SERVE_ROBOTS_TXT", Value: "true"},
						{Name: "TARGET", Value: target},
					},
					ReadinessProbe: &corev1.Probe{
						FailureThreshold: 3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Port: intstr.FromInt32(9090),
								Path: "/metrics",
							},
						},
					},
					Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:      ptr.To(int64(1000)),
						RunAsGroup:     ptr.To(int64(1000)),
						RunAsNonRoot:   ptr.To(true),
						Capabilities:   &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
				}},
			},
		}

		return nil
	})
	return err
}

// reconcileService ensures that the service exists
func (ir *IngressReconciler) reconcileService(ctx context.Context, req reconcile.Request) error {
	serv := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ia-" + req.Name,
			Namespace: ir.cfg.Namespace,
		},
	}

	labels := map[string]string{
		"app.kubernetes.io/instance": "anubis",
		"app.kubernetes.io/name":     "anubis",
		ManagedLabel:                 "true",
		OwningLabel:                  req.Namespace + "--" + req.Name,
	}

	_, err := controllerutil.CreateOrUpdate(ctx, ir.client, serv, func() error {
		serv.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       8080,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		}}

		serv.Spec.Selector = labels
		serv.Spec.Type = corev1.ServiceTypeClusterIP

		return nil
	})
	return err
}

// reconcileChildIngress reconciles the child (managed) Ingress
func (ir *IngressReconciler) reconcileChildIngress(ctx context.Context, origIngress *networkingv1.Ingress, req reconcile.Request) error {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ia-" + req.Name,
			Namespace: ir.cfg.Namespace,
		},
	}

	labels := map[string]string{
		"app.kubernetes.io/instance": "anubis",
		"app.kubernetes.io/name":     "anubis",
		ManagedLabel:                 "true",
		OwningLabel:                  req.Namespace + "--" + req.Name,
	}

	_, err := controllerutil.CreateOrUpdate(ctx, ir.client, ing, func() error {
		ing.Spec = *origIngress.Spec.DeepCopy()
		ing.ObjectMeta.Annotations = origIngress.ObjectMeta.DeepCopy().GetAnnotations()

		ing.Spec.IngressClassName = &ir.cfg.WrappedIngressClassName

		// Ensure our labels are set.
		if ing.ObjectMeta.Labels == nil {
			ing.ObjectMeta.Labels = make(map[string]string)
		}
		for k, v := range labels {
			ing.ObjectMeta.Labels[k] = v
		}

		// Ensure all hosts point to us instead of whatever was originally
		// set.
		backend := &networkingv1.IngressServiceBackend{
			Name: "ia-" + req.Name,
			Port: networkingv1.ServiceBackendPort{
				Name: "http",
			},
		}
		if ing.Spec.DefaultBackend != nil {
			ing.Spec.DefaultBackend.Service = backend
		}
		for i, r := range ing.Spec.Rules {
			if r.HTTP == nil {
				continue // TODO(jaredallard): Validate this case.
			}
			for j := range r.HTTP.Paths {
				ing.Spec.Rules[i].HTTP.Paths[j].Backend.Service = backend
			}
		}
		return nil
	})
	return err
}
