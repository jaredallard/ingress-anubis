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
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jaredallard/ingress-anubis/internal/config"
	"go.rgst.io/stencil/v2/pkg/slogext"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ManagedLabel is the label used to track what is managed by this
	// controller.
	ManagedLabel = "ingress-anubis.jaredallard.github.com/managed"

	// OwningLabel is the label used to store the owning ingress.
	OwningLabel = "ingress-anubis.jaredallard.github.com/owner"

	// FinalizerKey is the key to use for ingress-anubis's finalizer.
	FinalizerKey = "ingress-anubis.jaredallard.github.com/finalizer"
)

// IngressReconciler is the main reconciler of the controller. See
// [IngressReconciler.Reconcile] for more information.
type IngressReconciler struct {
	log    slogext.Logger
	cfg    *config.Config
	client crclient.Client
}

// mirrorStatus mirrors the status from a managed ingress to the owning
// ingressClass'd ingress
func (ir *IngressReconciler) mirrorStatus(ctx context.Context, ing *networkingv1.Ingress) (reconcile.Result, error) {
	targetIngKey, ok := ing.Labels[OwningLabel]
	if !ok {
		return reconcile.Result{}, nil
	}

	// TODO(jaredallard): This probably will break on any namespaces that
	// have '--' in the name.
	spl := strings.Split(targetIngKey, "--")
	if len(spl) != 2 {
		return reconcile.Result{},
			fmt.Errorf("failed to determine owner from owning label value %q", targetIngKey)
	}

	owningIng := &networkingv1.Ingress{}
	if err := ir.client.Get(ctx, crclient.ObjectKey{
		Namespace: spl[0],
		Name:      spl[1],
	}, owningIng); err != nil {
		return reconcile.Result{}, crclient.IgnoreNotFound(err)
	}

	patch := crclient.StrategicMergeFrom(owningIng.DeepCopy())
	owningIng.Status = ing.Status
	if err := ir.client.Status().Patch(ctx, owningIng, patch); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	return reconcile.Result{}, nil
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

	// Not controlled by us, only check to see if its a managed ingress
	// which we do want to handle for status mirroring purposes.
	if origIng.Spec.IngressClassName == nil || *origIng.Spec.IngressClassName != ir.cfg.IngressClassName {
		if origIng.Labels[ManagedLabel] == "true" {
			return ir.mirrorStatus(ctx, origIng)
		}

		return reconcile.Result{}, nil
	}

	if origIng.GetLabels()[ManagedLabel] == "true" {
		return reconcile.Result{}, reconcile.TerminalError(fmt.Errorf("attempted to reconcile ingress owned by self"))
	}

	log := ir.log.With(slog.String("name", req.Name), slog.String("namespace", req.Namespace))

	// Ingress was deleted, clean up resources.
	if !origIng.DeletionTimestamp.IsZero() {
		log.Info("ingress was deleted, pruning resources")

		if err := ir.deleteResources(ctx, req.Name); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to prune resources: %w", err)
		}

		// Remove the finalizer if it exists
		if slices.Contains(origIng.Finalizers, FinalizerKey) {
			patch := crclient.StrategicMergeFrom(origIng.DeepCopy())
			origIng.Finalizers = slices.Delete(origIng.Finalizers, slices.Index(origIng.Finalizers, FinalizerKey), 1)
			if err := ir.client.Patch(ctx, origIng, patch); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}

		log.Info("finished pruning resources and removed finalizer")

		return reconcile.Result{}, nil
	}

	log.Info("reconciling ingress")

	// If we don't have a finalizer set for us, add it.
	if !slices.Contains(origIng.Finalizers, FinalizerKey) {
		log.Info("adding finalizer")

		patch := crclient.StrategicMergeFrom(origIng.DeepCopy())
		origIng.Finalizers = append(origIng.Finalizers, FinalizerKey)
		if err := ir.client.Patch(ctx, origIng, patch); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}

		return reconcile.Result{Requeue: true}, nil
	}

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

	icfg, err := config.GetIngressConfigFromIngress(origIng)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileDeployment(ctx, target, icfg, req); err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileService(ctx, req); err != nil {
		return reconcile.Result{}, err
	}

	if err := ir.reconcileChildIngress(ctx, origIng, icfg, req); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// deleteResources cleans up all resources created by this controller,
// if they exist
func (ir *IngressReconciler) deleteResources(ctx context.Context, name string) error {
	meta := metav1.ObjectMeta{
		Name:      "ia-" + name,
		Namespace: ir.cfg.Namespace,
	}

	ing := &networkingv1.Ingress{}
	if err := ir.client.Get(ctx, crclient.ObjectKeyFromObject(&networkingv1.Ingress{ObjectMeta: meta}), ing); err == nil {
		if err := ir.client.Delete(ctx, ing); err != nil {
			return fmt.Errorf("failed to delete wrapped ingress: %w", err)
		}
	} else if err := crclient.IgnoreNotFound(err); err != nil {
		return fmt.Errorf("failed to check existence of wrapped ingress: %w", err)
	}

	svc := &corev1.Service{}
	if err := ir.client.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{ObjectMeta: meta}), svc); err == nil {
		if err := ir.client.Delete(ctx, svc); err != nil {
			return fmt.Errorf("failed to delete service: %w", err)
		}
	} else if err := crclient.IgnoreNotFound(err); err != nil {
		return fmt.Errorf("failed to check existence of service: %w", err)
	}

	dep := &appsv1.Deployment{}
	if err := ir.client.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{ObjectMeta: meta}), dep); err == nil {
		if err := ir.client.Delete(ctx, dep); err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	} else if err := crclient.IgnoreNotFound(err); err != nil {
		return fmt.Errorf("failed to check existence of deployment: %w", err)
	}

	return nil
}

// getTargetFromService returns a that can be used to communicate with
// the given service in isb from inside of Kubernetes.
func (ir *IngressReconciler) getTargetFromService(ctx context.Context, ns string,
	isb *networkingv1.IngressServiceBackend) (string, error) {
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

// getEnvFrom returns an EnvFrom block for the current ingress
// configuration
func (ir *IngressReconciler) getEnvFrom(icfg *config.IngressConfig) []corev1.EnvFromSource {
	envFrom := make([]corev1.EnvFromSource, 0)

	if ir.cfg.EnvFromCM != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: ir.cfg.EnvFromCM,
				},
			},
		})
	}

	if ir.cfg.EnvFromSec != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: ir.cfg.EnvFromSec,
				},
			},
		})
	}

	if icfg.EnvFromCM != nil {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: *icfg.EnvFromCM,
				},
			},
		})
	}

	if icfg.EnvFromSec != nil {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: *icfg.EnvFromSec,
				},
			},
		})
	}

	return envFrom
}

// getVolumeMounts returns the volume mounts for this instance
func (ir *IngressReconciler) getVolumeMounts() []corev1.VolumeMount {
	var r []corev1.VolumeMount

	//nolint:errcheck // Why: Best effort
	_ = json.Unmarshal([]byte(ir.cfg.VolumeMounts), &r)

	return r
}

// getVolumes returns the volumes for this instance
func (ir *IngressReconciler) getVolumes() []corev1.Volume {
	var r []corev1.Volume

	//nolint:errcheck // Why: Best effort
	_ = json.Unmarshal([]byte(ir.cfg.Volumes), &r)

	return r
}

// reconcileDeployment ensures that a deployment of anubis exists
func (ir *IngressReconciler) reconcileDeployment(ctx context.Context, target string,
	icfg *config.IngressConfig, req reconcile.Request) error {
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
		if dep.CreationTimestamp.IsZero() {
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}

		dep.Labels = labels

		// Only one replica is supported by anubis currently
		dep.Spec.Replicas = ptr.To(int32(1))
		dep.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}

		envVars := maps.Clone(ir.cfg.EnvironmentVariables)
		if envVars == nil {
			envVars = make(map[string]string)
		}

		// We override/set a few values controlled by us but also that have
		// their own annotation configuration values.
		envVars["BIND"] = ":8080"
		envVars["DIFFICULTY"] = strconv.Itoa(*icfg.Difficulty)
		envVars["METRICS_BIND"] = ":" + strconv.Itoa(int(*icfg.MetricsPort))
		envVars["SERVE_ROBOTS_TXT"] = strconv.FormatBool(*icfg.ServeRobotsTxt)
		envVars["TARGET"] = target
		envVars["OG_PASSTHROUGH"] = strconv.FormatBool(*icfg.OGPassthrough)

		cEnvVars := make([]corev1.EnvVar, 0, len(envVars))
		for k, v := range envVars {
			cEnvVars = append(cEnvVars, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}

		dep.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: ir.cfg.Annotations},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "main",
					Image: ir.cfg.AnubisImage + ":" + ir.cfg.AnubisVersion,
					Env:   cEnvVars,
					ReadinessProbe: &corev1.Probe{
						FailureThreshold: 3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								//nolint:gosec // Why: Not a possible overflow.
								Port: intstr.FromInt32(int32(*icfg.MetricsPort)),
								Path: "/metrics",
							},
						},
					},
					EnvFrom: ir.getEnvFrom(icfg),
					Ports: []corev1.ContainerPort{
						{Name: "http", ContainerPort: 8080},
						//nolint:gosec // Why: Not a possible overflow.
						{Name: "http-metrics", ContainerPort: int32(*icfg.MetricsPort)},
					},
					VolumeMounts: ir.getVolumeMounts(),
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						RunAsUser:                ptr.To(int64(1000)),
						RunAsGroup:               ptr.To(int64(1000)),
						RunAsNonRoot:             ptr.To(true),
						ReadOnlyRootFilesystem:   ptr.To(true),
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
				}},
				Volumes: ir.getVolumes(),
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
func (ir *IngressReconciler) reconcileChildIngress(ctx context.Context, origIng *networkingv1.Ingress,
	icfg *config.IngressConfig, req reconcile.Request) error {
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
		ing.Spec = *origIng.Spec.DeepCopy()
		ing.Annotations = origIng.DeepCopy().GetAnnotations()

		if icfg.IngressClass != nil {
			ing.Spec.IngressClassName = icfg.IngressClass
		} else {
			ing.Spec.IngressClassName = &ir.cfg.WrappedIngressClassName
		}

		// Ensure our labels are set.
		if ing.Labels == nil {
			ing.Labels = make(map[string]string)
		}
		maps.Insert(ing.Labels, maps.All(labels))

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
