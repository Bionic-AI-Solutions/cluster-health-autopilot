// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconciler reconciles a ClusterHealthAutopilot CR into the
// workloads the existing chart-managed install also produces:
// ServiceAccount, watcher Deployment, diagnose CronJob, remediate
// CronJob.
//
// Phase 1b deliberately ships *only* the lifecycle manager — the
// controller never runs probes / analyzers / fixers itself. The
// existing `cmd/cha` binary running inside the watcher Deployment is
// the loop. The operator's job is to make sure the workload is
// shaped according to the CR.
//
// The Reconcile contract:
//
//  1. Fetch the CR.
//  2. Defer a status update (so condition updates land even on
//     mid-reconcile errors).
//  3. Build the desired child objects with the pure-function
//     builders in builders.go.
//  4. CreateOrUpdate each: if it exists, patch toward desired; if
//     it doesn't, create. Disabled subresources (Watcher.Enabled=
//     false) get explicit Delete calls.
//  5. Set Ready / WatcherRunning conditions from the live
//     Deployment / CronJob state.
//
// Server-side-apply would be a cleaner long-term path, but
// controller-runtime's SSA support requires a managed-fields
// migration when adopting the operator on top of a chart-managed
// install. CreateOrUpdate keeps the cutover boring: existing chart
// installs aren't disturbed unless an operator explicitly creates
// a ClusterHealthAutopilot CR, in which case the controller takes
// over the named resources.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is the controller-runtime entrypoint. Signature is
// pinned by the framework.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("clusterhealthautopilot", req.NamespacedName)

	// 1. Fetch the CR. Tolerate NotFound — that's the normal
	// post-delete reconcile (owner-refs have already cleaned up
	// children).
	var cr chav1alpha1.ClusterHealthAutopilot
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetch CR: %w", err)
	}

	// Validate before doing any work. Refusing here short-circuits
	// the rest and surfaces the problem via Ready=False.
	if cr.Spec.Image.Tag == "" {
		r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
			"spec.image.tag is required and must be non-empty")
		return ctrl.Result{}, r.updateStatus(ctx, &cr)
	}

	// 2. Reconcile each subresource. Collect errors so the
	// status update reflects the partial state honestly.
	var firstErr error
	track := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	track(r.reconcileServiceAccount(ctx, &cr))
	track(r.reconcileWatcher(ctx, &cr))
	track(r.reconcileDiagnose(ctx, &cr))
	track(r.reconcileRemediate(ctx, &cr))

	// 3. Compute Ready + WatcherRunning conditions from the
	// observed cluster state.
	r.computeConditions(ctx, &cr, firstErr)

	if err := r.updateStatus(ctx, &cr); err != nil {
		log.Error(err, "status update failed")
	}
	return ctrl.Result{}, firstErr
}

// reconcileServiceAccount ensures the SA the workloads reference
// exists. The owner-ref makes the CR's deletion cascade clean it up.
func (r *Reconciler) reconcileServiceAccount(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	// BYO ServiceAccount: when the CR pins spec.serviceAccountName the
	// operator must NOT create or own that SA — it belongs to the
	// caller (typically the chart's reader-bound SA, which already
	// carries the probe RBAC the watcher needs). Owning it would graft
	// an owner-ref onto a pre-existing SA and garbage-collect it when
	// the CR is deleted. Operator-managed watchers get probe RBAC ONLY
	// via this BYO path today; the operator does not yet provision its
	// own reader ClusterRoleBinding (tracked for Phase 1c).
	if cr.Spec.ServiceAccountName != "" {
		return nil
	}
	desired := BuildServiceAccount(cr)
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desired, func(current client.Object) {
		// SAs are essentially metadata-only; we don't patch any
		// runtime fields. Tag mismatch == no-op.
		c := current.(*corev1.ServiceAccount)
		mergeLabels(&c.ObjectMeta, desired.Labels)
	})
}

// reconcileWatcher creates / updates / deletes the watcher
// Deployment based on Spec.Watcher.
func (r *Reconciler) reconcileWatcher(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	desired := BuildWatcherDeployment(cr)
	name := NamesFor(cr).Watcher
	if desired == nil {
		return r.deleteIfExists(ctx, &appsv1.Deployment{}, cr.Namespace, name)
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desired, func(current client.Object) {
		c := current.(*appsv1.Deployment)
		c.Spec.Replicas = desired.Spec.Replicas
		c.Spec.Template = desired.Spec.Template
		c.Spec.Strategy = desired.Spec.Strategy
		mergeLabels(&c.ObjectMeta, desired.Labels)
	})
}

// reconcileDiagnose creates / updates / deletes the diagnose CronJob.
func (r *Reconciler) reconcileDiagnose(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	desired := BuildDiagnoseCronJob(cr)
	name := NamesFor(cr).Diagnose
	if desired == nil {
		return r.deleteIfExists(ctx, &batchv1.CronJob{}, cr.Namespace, name)
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desired, func(current client.Object) {
		c := current.(*batchv1.CronJob)
		c.Spec.Schedule = desired.Spec.Schedule
		c.Spec.JobTemplate = desired.Spec.JobTemplate
		c.Spec.ConcurrencyPolicy = desired.Spec.ConcurrencyPolicy
		c.Spec.SuccessfulJobsHistoryLimit = desired.Spec.SuccessfulJobsHistoryLimit
		c.Spec.FailedJobsHistoryLimit = desired.Spec.FailedJobsHistoryLimit
		mergeLabels(&c.ObjectMeta, desired.Labels)
	})
}

// reconcileRemediate creates / updates / deletes the remediate
// CronJob. Same shape as diagnose.
func (r *Reconciler) reconcileRemediate(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	desired := BuildRemediateCronJob(cr)
	name := NamesFor(cr).Remediate
	if desired == nil {
		return r.deleteIfExists(ctx, &batchv1.CronJob{}, cr.Namespace, name)
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desired, func(current client.Object) {
		c := current.(*batchv1.CronJob)
		c.Spec.Schedule = desired.Spec.Schedule
		c.Spec.JobTemplate = desired.Spec.JobTemplate
		c.Spec.ConcurrencyPolicy = desired.Spec.ConcurrencyPolicy
		c.Spec.SuccessfulJobsHistoryLimit = desired.Spec.SuccessfulJobsHistoryLimit
		c.Spec.FailedJobsHistoryLimit = desired.Spec.FailedJobsHistoryLimit
		mergeLabels(&c.ObjectMeta, desired.Labels)
	})
}

// createOrUpdate is the local CreateOrUpdate variant. Standard
// controller-runtime ctrl.CreateOrUpdate would also work; we keep
// our own so the mutate signature is typed by us (no reflection in
// the mutate callback).
func (r *Reconciler) createOrUpdate(ctx context.Context, desired client.Object, mutate func(client.Object)) error {
	key := client.ObjectKeyFromObject(desired)
	current := desired.DeepCopyObject().(client.Object)
	err := r.Get(ctx, key, current)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get %T %s: %w", desired, key, err)
		}
		// Create path.
		if cerr := r.Create(ctx, desired); cerr != nil && !apierrors.IsAlreadyExists(cerr) {
			return fmt.Errorf("create %T %s: %w", desired, key, cerr)
		}
		return nil
	}
	// Update path — apply the mutate callback to the live object
	// and Update it.
	mutate(current)
	if uerr := r.Update(ctx, current); uerr != nil {
		return fmt.Errorf("update %T %s: %w", desired, key, uerr)
	}
	return nil
}

// deleteIfExists removes a child resource that's been disabled in
// the spec. Tolerates NotFound.
func (r *Reconciler) deleteIfExists(ctx context.Context, obj client.Object, namespace, name string) error {
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return client.IgnoreNotFound(r.Delete(ctx, obj))
}

// computeConditions sets the two stable conditions Phase 1 documents
// in the CRD's printer columns: Ready + WatcherRunning.
func (r *Reconciler) computeConditions(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot, firstErr error) {
	// WatcherRunning — read the live Deployment's status.
	var watcherStatus metav1.ConditionStatus
	var watcherReason, watcherMsg string
	if cr.Spec.Watcher == nil || !cr.Spec.Watcher.Enabled {
		watcherStatus = metav1.ConditionFalse
		watcherReason = "Disabled"
		watcherMsg = "spec.watcher.enabled is false"
	} else {
		var dep appsv1.Deployment
		err := r.Get(ctx, types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      NamesFor(cr).Watcher,
		}, &dep)
		switch {
		case apierrors.IsNotFound(err):
			watcherStatus = metav1.ConditionFalse
			watcherReason = "DeploymentMissing"
			watcherMsg = "watcher Deployment not found"
		case err != nil:
			watcherStatus = metav1.ConditionUnknown
			watcherReason = "GetFailed"
			watcherMsg = err.Error()
		case dep.Status.AvailableReplicas > 0:
			watcherStatus = metav1.ConditionTrue
			watcherReason = "AvailableReplicasPositive"
			watcherMsg = fmt.Sprintf("availableReplicas=%d/%d",
				dep.Status.AvailableReplicas, *dep.Spec.Replicas)
		default:
			watcherStatus = metav1.ConditionFalse
			watcherReason = "NoAvailableReplicas"
			watcherMsg = "watcher Deployment has no available replicas"
		}
	}
	setCondition(&cr.Status, chav1alpha1.ConditionWatcherRunning,
		watcherStatus, watcherReason, watcherMsg, cr.Generation)

	// Ready — true iff all subresources reconciled without error.
	readyStatus := metav1.ConditionTrue
	readyReason := "Reconciled"
	readyMsg := "all subresources reconciled"
	if firstErr != nil {
		readyStatus = metav1.ConditionFalse
		readyReason = "ReconcileError"
		readyMsg = firstErr.Error()
	}
	setCondition(&cr.Status, chav1alpha1.ConditionReady,
		readyStatus, readyReason, readyMsg, cr.Generation)

	cr.Status.ObservedGeneration = cr.Generation
}

// markReady is the validation-short-circuit helper.
func (r *Reconciler) markReady(cr *chav1alpha1.ClusterHealthAutopilot, status metav1.ConditionStatus, reason, msg string) {
	setCondition(&cr.Status, chav1alpha1.ConditionReady, status, reason, msg, cr.Generation)
}

// updateStatus pushes the in-memory status back to the apiserver.
func (r *Reconciler) updateStatus(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	return r.Status().Update(ctx, cr)
}

// SetupWithManager registers the reconciler with the controller-runtime
// manager. Owns() the four child resource types so changes to them
// requeue the parent CR.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chav1alpha1.ClusterHealthAutopilot{}, builder.WithPredicates()).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}

// setCondition upserts the named condition on the status.
func setCondition(s *chav1alpha1.ClusterHealthAutopilotStatus, condType string, status metav1.ConditionStatus, reason, msg string, gen int64) {
	now := metav1.Now()
	for i := range s.Conditions {
		if s.Conditions[i].Type == condType {
			c := &s.Conditions[i]
			if c.Status != status {
				c.LastTransitionTime = now
			}
			c.Status = status
			c.Reason = reason
			c.Message = msg
			c.ObservedGeneration = gen
			return
		}
	}
	s.Conditions = append(s.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: now,
		ObservedGeneration: gen,
	})
}

// mergeLabels copies any controller-managed labels onto the live
// object so labels added in builders.go (managed-by, role, etc.)
// stay present after Update.
func mergeLabels(meta *metav1.ObjectMeta, desired map[string]string) {
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}
	for k, v := range desired {
		meta.Labels[k] = v
	}
}

// controllerutilSetOwnerRef sets the CR as the owner of the child.
// Wrapper isolates the controllerutil import to one place.
func controllerutilSetOwnerRef(cr *chav1alpha1.ClusterHealthAutopilot, child client.Object, scheme *runtime.Scheme) error {
	return controllerutilHelper(cr, child, scheme)
}
