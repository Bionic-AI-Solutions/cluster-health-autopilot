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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	// 1b. Finalizer / deletion handling (Phase 1c).
	//
	// The per-CR reader ClusterRoleBinding the operator provisions is
	// CLUSTER-SCOPED. Kubernetes' garbage collector does NOT honor an
	// ownerRef from a namespaced parent to a cluster-scoped child — it
	// drops the ref and orphans the binding. So a finalizer drives
	// explicit cleanup before the CR is GC'd.
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, chav1alpha1.FinalizerOperatorRBAC) {
			if err := r.finalizeReaderRBAC(ctx, &cr); err != nil {
				return ctrl.Result{}, fmt.Errorf("finalize reader RBAC: %w", err)
			}
			controllerutil.RemoveFinalizer(&cr, chav1alpha1.FinalizerOperatorRBAC)
			if err := r.Update(ctx, &cr); err != nil {
				return ctrl.Result{}, fmt.Errorf("clear finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}
	// Ensure the finalizer is in place BEFORE reconciling the binding
	// (so a CR delete between create and the next reconcile still
	// gets the cleanup pass).
	if controllerutil.AddFinalizer(&cr, chav1alpha1.FinalizerOperatorRBAC) {
		if err := r.Update(ctx, &cr); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		// The Update re-queues us with a fresh ResourceVersion; the
		// continuation here uses the in-memory cr we already mutated.
	}

	// Validate before doing any work. Refusing here short-circuits
	// the rest and surfaces the problem via Ready=False.
	if cr.Spec.Image.Tag == "" {
		r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
			"spec.image.tag is required and must be non-empty")
		return ctrl.Result{}, r.updateStatus(ctx, &cr)
	}
	if AIEnabled(&cr) {
		// Mirror the chart's `required` directives for `cha.aiArgs`.
		// Better to fail fast here than to ship the aiwatch with an
		// empty --ai-endpoint / --ai-model and CrashLoopBackoff.
		if cr.Spec.AI.Endpoint == "" {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.ai.endpoint is required when spec.ai.enabled=true")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
		if cr.Spec.AI.Model == "" {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.ai.model is required when spec.ai.enabled=true")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
		if m := cr.Spec.AI.Memory; m != nil && m.Enabled &&
			(m.Embeddings == nil || m.Embeddings.Model == "") {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.ai.memory.embeddings.model is required when spec.ai.memory.enabled=true")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
	}
	if MemoryEnabled(&cr) {
		// Validate the storage size up front — the builder's fallback
		// would silently swap a typo'd "5gb" for the 5Gi default,
		// which is exactly the kind of "looked right, did the wrong
		// thing" failure we want to make loud.
		if s := cr.Spec.AI.Memory.Storage; s != nil && s.Size != "" {
			if _, err := parseQuantity(s.Size); err != nil {
				r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
					"spec.ai.memory.storage.size: "+err.Error())
				return ctrl.Result{}, r.updateStatus(ctx, &cr)
			}
		}
	}
	if ApprovalIngressEnabled(&cr) {
		// The chart's approval-server-ingress.yaml `required`s host;
		// match that here so an operator-managed install fails fast
		// rather than producing a hostless Ingress (which would land
		// but route nothing).
		if cr.Spec.Approval.Ingress.Host == "" {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.approval.ingress.host is required when spec.approval.ingress.enabled=true")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
	}
	if ApprovalNetworkPolicyEnabled(&cr) {
		// The NetworkPolicy is fail-closed: once it selects the
		// approval-server pods, all ingress not matching an allow rule
		// is dropped. An empty gatewayNamespaceSelector would either
		// match nothing (blocking ALL traffic, including legit
		// oauth2-proxy → approval clicks) or — if rendered as an empty
		// selector — match EVERY namespace (defeating the control).
		// There is intentionally no default; fail fast so the operator
		// declares the gateway namespace's labels explicitly.
		if len(cr.Spec.Approval.NetworkPolicy.GatewayNamespaceSelector) == 0 {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.approval.networkPolicy.gatewayNamespaceSelector is required when "+
					"spec.approval.networkPolicy.enabled=true (no safe default exists; "+
					"set it to the gateway/oauth2-proxy namespace's labels, e.g. "+
					"{kubernetes.io/metadata.name: <gateway-namespace>})")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
	}
	if ApprovalEnabled(&cr) {
		// The in-memory replay store is per-replica: each pod holds an
		// independent set of consumed JTIs. Running >1 replica means
		// an approval click routed to replica B is not deduped by
		// replica A — a replay attack becomes possible for any JTI
		// that only one replica has seen. The configmap backend shares
		// state via K8s and is safe for replicas > 1. Fail fast so the
		// operator never silently creates a split-brain approval fleet.
		if approvalStoreBackend(&cr) == "inmemory" && approvalReplicas(&cr) > 1 {
			r.markReady(&cr, metav1.ConditionFalse, "InvalidSpec",
				"spec.approval.replicas must be 1 when spec.approval.store.backend=inmemory (default); "+
					"set store.backend=configmap to run multiple replicas safely")
			return ctrl.Result{}, r.updateStatus(ctx, &cr)
		}
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
	track(r.reconcileReaderRBAC(ctx, &cr))
	track(r.reconcileWatcher(ctx, &cr))
	track(r.reconcileDiagnose(ctx, &cr))
	track(r.reconcileRemediate(ctx, &cr))
	track(r.reconcileAIWatch(ctx, &cr))
	track(r.reconcileQdrant(ctx, &cr))
	track(r.reconcileApprovalServer(ctx, &cr))

	// 3. Compute Ready + WatcherRunning + ReaderRBACReady conditions
	// from the observed cluster state.
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

// reconcileReaderRBAC ensures the operator-provisioned reader
// ClusterRole + per-CR ClusterRoleBinding both exist. The role is
// shared across every CR in the cluster; the binding is per-CR and
// references the CR's ServiceAccount.
//
// Neither object carries an ownerRef back to the CR — they're cluster-
// scoped, and Kubernetes drops namespaced ownerRefs on cluster-scoped
// children. Cleanup runs through the finalizer (see finalizeReaderRBAC).
func (r *Reconciler) reconcileReaderRBAC(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	// 1. Shared ClusterRole — create-or-update idempotently. Survives
	//    every CR delete on purpose (other CRs may still reference it).
	desiredRole := BuildReaderClusterRole()
	if err := r.createOrUpdate(ctx, desiredRole, func(current client.Object) {
		c := current.(*rbacv1.ClusterRole)
		c.Rules = desiredRole.Rules
		mergeLabels(&c.ObjectMeta, desiredRole.Labels)
	}); err != nil {
		return fmt.Errorf("reader ClusterRole: %w", err)
	}

	// 2. Per-CR ClusterRoleBinding — Subject must point at the
	//    operator-managed SA (or the BYO-SA when spec.serviceAccountName
	//    is set). Re-derived from the CR every reconcile so a CR Spec
	//    change to the SA name is picked up.
	desiredBinding := BuildReaderClusterRoleBinding(cr)
	return r.createOrUpdate(ctx, desiredBinding, func(current client.Object) {
		c := current.(*rbacv1.ClusterRoleBinding)
		c.RoleRef = desiredBinding.RoleRef
		c.Subjects = desiredBinding.Subjects
		mergeLabels(&c.ObjectMeta, desiredBinding.Labels)
	})
}

// finalizeReaderRBAC deletes the per-CR cluster-scoped ClusterRoleBindings
// before the CR is garbage-collected. The shared ClusterRoles stay — they
// may be in use by OTHER CRs in the cluster, and a leftover unbound role
// is harmless on cleanup.
//
// Phase 2c-B extended this to ALSO clean up the per-CR approval-fixer
// binding when AI/approval is enabled. Both bindings are cluster-scoped
// and labeled with ManagedByCRLabel, so the same defense-in-depth check
// applies.
func (r *Reconciler) finalizeReaderRBAC(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	bindingNames := []string{
		ReaderClusterRoleBindingName(cr),
		ApprovalFixerClusterRoleBindingName(cr),
	}
	for _, name := range bindingNames {
		binding := &rbacv1.ClusterRoleBinding{}
		err := r.Get(ctx, types.NamespacedName{Name: name}, binding)
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		// Defense in depth: only delete a binding we actually labeled
		// ourselves. Stops a manual `kubectl create clusterrolebinding`
		// from being garbage-collected when the user happens to pick
		// the same name.
		if binding.Labels[ManagedByCRLabel] != cr.Name ||
			binding.Labels[ManagedByCRNamespaceLabel] != cr.Namespace {
			continue
		}
		if err := client.IgnoreNotFound(r.Delete(ctx, binding)); err != nil {
			return err
		}
	}

	// P1.9(c) — clean up the cross-namespace approval events Role +
	// RoleBinding. When spec.approval.auditNamespace points at a
	// namespace OTHER than the CR's own, the "<name>-events" Role +
	// RoleBinding are created there WITHOUT an ownerRef (cross-namespace
	// ownerRefs are illegal), so Kubernetes GC never reaps them. The
	// disable-while-alive teardown handles them, but a straight CR
	// delete skips that path — they leaked for the cluster's lifetime
	// until now. (When auditNamespace == cr.Namespace the objects are
	// owner-ref'd and GC handles them; this delete is then a harmless
	// no-op — NotFound is ignored.)
	if auditNS := ApprovalAuditNamespace(cr); auditNS != cr.Namespace {
		eventsName := ApprovalServerName(cr) + "-events"
		eventsObjs := []client.Object{
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: eventsName, Namespace: auditNS}},
			&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: eventsName, Namespace: auditNS}},
		}
		for _, obj := range eventsObjs {
			if err := client.IgnoreNotFound(r.Delete(ctx, obj)); err != nil {
				return err
			}
		}
	}
	return nil
}

// reconcileWatcher creates / updates / deletes the watcher
// Deployment based on Spec.Watcher, plus the class-E webhook
// receiver Service when spec.watcher.triggers.webhook.serviceEnabled.
func (r *Reconciler) reconcileWatcher(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	desired := BuildWatcherDeployment(cr)
	name := NamesFor(cr).Watcher
	svcName := WebhookServiceNameFor(cr)
	if desired == nil {
		// Watcher disabled — tear down the Deployment AND the webhook
		// Service (the latter is harmless if absent).
		if err := r.deleteIfExists(ctx, &appsv1.Deployment{}, cr.Namespace, name); err != nil {
			return err
		}
		return r.deleteIfExists(ctx, &corev1.Service{}, cr.Namespace, svcName)
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, desired, func(current client.Object) {
		c := current.(*appsv1.Deployment)
		c.Spec.Replicas = desired.Spec.Replicas
		c.Spec.Template = desired.Spec.Template
		c.Spec.Strategy = desired.Spec.Strategy
		mergeLabels(&c.ObjectMeta, desired.Labels)
	}); err != nil {
		return err
	}

	// P1.5 — webhook receiver Service. Reconciled separately so a
	// serviceEnabled flip (off/on/port-change) doesn't perturb the
	// Deployment rollout. Returns nil when the receiver isn't
	// listening or serviceEnabled=false; tears down the Service when
	// it flips off. Mirrors the aiwatch metrics-Service pattern AND
	// the chart's watcher-webhook-service.yaml semantics.
	desiredSvc := BuildWatcherWebhookService(cr)
	if desiredSvc == nil {
		return r.deleteIfExists(ctx, &corev1.Service{}, cr.Namespace, svcName)
	}
	if err := controllerutilSetOwnerRef(cr, desiredSvc, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desiredSvc, func(current client.Object) {
		c := current.(*corev1.Service)
		// Don't touch clusterIP (immutable). Tracking ports +
		// selector + labels.
		c.Spec.Ports = desiredSvc.Spec.Ports
		c.Spec.Selector = desiredSvc.Spec.Selector
		mergeLabels(&c.ObjectMeta, desiredSvc.Labels)
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

// reconcileAIWatch creates / updates / deletes the CHA-com aiwatch
// Deployment based on Spec.AI. Mirrors reconcileWatcher.
//
// Phase 2: the aiwatch container is the AI-companion to the OSS
// watcher; it polls the same merged probe+analyzer catalog and fires
// recommendation-only AI tiers against new diagnostics. Enabling AI
// is purely additive — the OSS watcher Deployment and the diagnose /
// remediate CronJobs keep running independently.
func (r *Reconciler) reconcileAIWatch(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	desired := BuildAIWatchDeployment(cr)
	name := NamesFor(cr).AIWatch
	if desired == nil {
		// AI disabled — tear down the Deployment AND the metrics
		// Service (the latter is harmless if absent).
		if err := r.deleteIfExists(ctx, &appsv1.Deployment{}, cr.Namespace, name); err != nil {
			return err
		}
		return r.deleteIfExists(ctx, &corev1.Service{}, cr.Namespace, name+"-metrics")
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	if err := r.createOrUpdate(ctx, desired, func(current client.Object) {
		c := current.(*appsv1.Deployment)
		c.Spec.Replicas = desired.Spec.Replicas
		c.Spec.Template = desired.Spec.Template
		c.Spec.Strategy = desired.Spec.Strategy
		mergeLabels(&c.ObjectMeta, desired.Labels)
	}); err != nil {
		return err
	}

	// Phase 3.D — Metrics Service. Reconciled separately so a
	// metrics flip (off/on/port-change) doesn't perturb the
	// Deployment rolloput. Returns nil when metrics aren't
	// configured; tears down the Service when they flip off.
	desiredSvc := BuildAIWatchMetricsService(cr)
	svcName := name + "-metrics"
	if desiredSvc == nil {
		return r.deleteIfExists(ctx, &corev1.Service{}, cr.Namespace, svcName)
	}
	if err := controllerutilSetOwnerRef(cr, desiredSvc, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desiredSvc, func(current client.Object) {
		c := current.(*corev1.Service)
		// Don't touch clusterIP (immutable). Tracking ports +
		// selector + labels.
		c.Spec.Ports = desiredSvc.Spec.Ports
		c.Spec.Selector = desiredSvc.Spec.Selector
		mergeLabels(&c.ObjectMeta, desiredSvc.Labels)
	})
}

// reconcileApprovalServer creates / updates / deletes the
// approval-server stack (SA + signing-key Secret + Deployment +
// Service + fixer ClusterRole/Binding + 3 namespaced Roles+Bindings)
// based on spec.approval. Mirrors the chart's
// templates/approval-server-*.yaml. Phase 2c-B.
//
// The signing-key Secret is operator-generated (Ed25519) and
// idempotent — once created, never regenerated. Replicas of the
// operator reading the same Secret produce the same JWT verification
// behavior.
func (r *Reconciler) reconcileApprovalServer(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	ns := cr.Namespace
	name := ApprovalServerName(cr)

	desiredSA := BuildApprovalServerServiceAccount(cr)
	desiredSvc := BuildApprovalServerService(cr)
	desiredDep := BuildApprovalServerDeployment(cr)

	if desiredSA == nil {
		// Approval disabled — tear down everything we manage. Signing-
		// key Secret is intentionally NOT deleted (Secret restore-on-
		// re-enable would otherwise invalidate every outstanding
		// approval JWT; users who want to rotate explicitly should
		// kubectl delete the Secret themselves). The shared fixer
		// ClusterRole is preserved like the reader role — other CRs
		// may still bind to it. The per-CR fixer ClusterRoleBinding
		// is cleaned up by the finalizer pass when the CR is GC'd.
		teardownList := []struct {
			obj client.Object
			n   string
			ns  string
		}{
			{&networkingv1.Ingress{}, name, ns},
			{&networkingv1.NetworkPolicy{}, name, ns},
			{&appsv1.Deployment{}, name, ns},
			{&corev1.Service{}, name, ns},
			{&corev1.ServiceAccount{}, name, ns},
			{&rbacv1.Role{}, name + "-signing-reader", ns},
			{&rbacv1.RoleBinding{}, name + "-signing-reader", ns},
			{&rbacv1.Role{}, name + "-events", ApprovalAuditNamespace(cr)},
			{&rbacv1.RoleBinding{}, name + "-events", ApprovalAuditNamespace(cr)},
			{&rbacv1.Role{}, name + "-stores", ns},
			{&rbacv1.RoleBinding{}, name + "-stores", ns},
			{&rbacv1.ClusterRoleBinding{}, ApprovalFixerClusterRoleBindingName(cr), ""},
		}
		for _, item := range teardownList {
			if err := r.deleteIfExists(ctx, item.obj, item.ns, item.n); err != nil {
				return fmt.Errorf("teardown %T %s: %w", item.obj, item.n, err)
			}
		}
		return nil
	}

	// Set owner refs on namespaced children (cluster-scoped objects
	// don't take ownerRefs from a namespaced parent — they're cleaned
	// up by the finalizer instead).
	for _, o := range []client.Object{desiredSA, desiredSvc, desiredDep} {
		if err := controllerutilSetOwnerRef(cr, o, r.Scheme); err != nil {
			return err
		}
	}

	// 1. Signing-key Secret — generate-once, then leave alone.
	if err := r.reconcileSigningKeySecret(ctx, cr); err != nil {
		return fmt.Errorf("signing-key Secret: %w", err)
	}

	// 2. ServiceAccount.
	if err := r.createOrUpdate(ctx, desiredSA, func(current client.Object) {
		c := current.(*corev1.ServiceAccount)
		mergeLabels(&c.ObjectMeta, desiredSA.Labels)
	}); err != nil {
		return fmt.Errorf("approval SA: %w", err)
	}

	// 3. Shared fixer ClusterRole — idempotent across all CRs.
	fixerRole := BuildApprovalFixerClusterRole()
	if err := r.createOrUpdate(ctx, fixerRole, func(current client.Object) {
		c := current.(*rbacv1.ClusterRole)
		c.Rules = fixerRole.Rules
		mergeLabels(&c.ObjectMeta, fixerRole.Labels)
	}); err != nil {
		return fmt.Errorf("approval fixer ClusterRole: %w", err)
	}

	// 4. Per-CR fixer ClusterRoleBinding.
	fixerBinding := BuildApprovalFixerClusterRoleBinding(cr)
	if err := r.createOrUpdate(ctx, fixerBinding, func(current client.Object) {
		c := current.(*rbacv1.ClusterRoleBinding)
		c.RoleRef = fixerBinding.RoleRef
		c.Subjects = fixerBinding.Subjects
		mergeLabels(&c.ObjectMeta, fixerBinding.Labels)
	}); err != nil {
		return fmt.Errorf("approval fixer binding: %w", err)
	}

	// 5. Namespaced Role/RoleBinding triplet.
	for _, pair := range []struct {
		role    *rbacv1.Role
		binding *rbacv1.RoleBinding
		label   string
	}{
		{BuildApprovalSigningReaderRole(cr), BuildApprovalSigningReaderRoleBinding(cr), "signing-reader"},
		{BuildApprovalEventsRole(cr), BuildApprovalEventsRoleBinding(cr), "events"},
		{BuildApprovalStoresRole(cr), BuildApprovalStoresRoleBinding(cr), "stores"},
	} {
		if pair.role == nil {
			// stores-RBAC is gated on configmap backend; events Role
			// is always non-nil but defensive.
			continue
		}
		// Owner-ref the Role + Binding to the CR (same namespace
		// when AuditNamespace defaults; if user pinned a different
		// AuditNamespace cross-namespace ownerRef would fail — the
		// next conditional handles that).
		if pair.role.Namespace == cr.Namespace {
			if err := controllerutilSetOwnerRef(cr, pair.role, r.Scheme); err != nil {
				return err
			}
			if err := controllerutilSetOwnerRef(cr, pair.binding, r.Scheme); err != nil {
				return err
			}
		}
		if err := r.createOrUpdate(ctx, pair.role, func(current client.Object) {
			c := current.(*rbacv1.Role)
			c.Rules = pair.role.Rules
			mergeLabels(&c.ObjectMeta, pair.role.Labels)
		}); err != nil {
			return fmt.Errorf("approval %s Role: %w", pair.label, err)
		}
		if err := r.createOrUpdate(ctx, pair.binding, func(current client.Object) {
			c := current.(*rbacv1.RoleBinding)
			c.RoleRef = pair.binding.RoleRef
			c.Subjects = pair.binding.Subjects
			mergeLabels(&c.ObjectMeta, pair.binding.Labels)
		}); err != nil {
			return fmt.Errorf("approval %s RoleBinding: %w", pair.label, err)
		}
	}

	// 6. Service.
	if err := r.createOrUpdate(ctx, desiredSvc, func(current client.Object) {
		c := current.(*corev1.Service)
		c.Spec.Ports = desiredSvc.Spec.Ports
		c.Spec.Selector = desiredSvc.Spec.Selector
		mergeLabels(&c.ObjectMeta, desiredSvc.Labels)
	}); err != nil {
		return fmt.Errorf("approval Service: %w", err)
	}

	// 7. Deployment.
	if err := r.createOrUpdate(ctx, desiredDep, func(current client.Object) {
		c := current.(*appsv1.Deployment)
		c.Spec.Replicas = desiredDep.Spec.Replicas
		c.Spec.Template = desiredDep.Spec.Template
		c.Spec.Strategy = desiredDep.Spec.Strategy
		mergeLabels(&c.ObjectMeta, desiredDep.Labels)
	}); err != nil {
		return fmt.Errorf("approval Deployment: %w", err)
	}

	// 8. Optional NetworkPolicy (P2.6b). Restricts ingress to the
	// approval-server pods to the gateway/oauth2-proxy namespace,
	// closing the X-Forwarded-User header-forgery bypass. Disabled →
	// tear down a stale NetworkPolicy left from a previous spec.
	desiredNetpol := BuildApprovalServerNetworkPolicy(cr)
	if desiredNetpol == nil {
		if err := r.deleteIfExists(ctx, &networkingv1.NetworkPolicy{}, ns, name); err != nil {
			return fmt.Errorf("teardown approval NetworkPolicy: %w", err)
		}
	} else {
		if err := controllerutilSetOwnerRef(cr, desiredNetpol, r.Scheme); err != nil {
			return err
		}
		if err := r.createOrUpdate(ctx, desiredNetpol, func(current client.Object) {
			c := current.(*networkingv1.NetworkPolicy)
			c.Spec = desiredNetpol.Spec
			mergeLabels(&c.ObjectMeta, desiredNetpol.Labels)
		}); err != nil {
			return fmt.Errorf("approval NetworkPolicy: %w", err)
		}
	}

	// 9. Optional Ingress (Phase 2c-C). Disabled → tear down a stale
	// Ingress object if one is left from a previous-spec.
	desiredIng := BuildApprovalServerIngress(cr)
	if desiredIng == nil {
		return r.deleteIfExists(ctx, &networkingv1.Ingress{}, ns, name)
	}
	if err := controllerutilSetOwnerRef(cr, desiredIng, r.Scheme); err != nil {
		return err
	}
	return r.createOrUpdate(ctx, desiredIng, func(current client.Object) {
		c := current.(*networkingv1.Ingress)
		c.Spec = desiredIng.Spec
		// Annotations carry user-provided cert-manager / oauth2-proxy
		// configuration — preserve any user-added annotations the
		// operator doesn't manage by merging.
		if c.Annotations == nil {
			c.Annotations = map[string]string{}
		}
		for k, v := range desiredIng.Annotations {
			c.Annotations[k] = v
		}
		mergeLabels(&c.ObjectMeta, desiredIng.Labels)
	})
}

// reconcileSigningKeySecret creates the Ed25519 signing-key Secret on
// the first reconcile and leaves it alone after. The operator MUST NOT
// regenerate the keypair on subsequent reconciles — every outstanding
// approval JWT (signed by the previous key) would become unverifiable.
func (r *Reconciler) reconcileSigningKeySecret(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	name := ApprovalSigningKeySecretName(cr)
	var existing corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: name}, &existing)
	if err == nil {
		// Already exists — never touch the key bytes. Refresh labels
		// only when they've actually drifted, avoiding a spurious
		// Update (and the resulting resourceVersion churn) every
		// reconcile. This is especially important because the operator
		// may run with multiple leader-election candidates; a no-op
		// Update on every reconcile adds noise to audit logs and can
		// trigger unnecessary watch events.
		desired := CommonLabels(cr, "approval-server")
		if labelsUpToDate(existing.Labels, desired) {
			return nil
		}
		mergeLabels(&existing.ObjectMeta, desired)
		return r.Update(ctx, &existing)
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get signing Secret: %w", err)
	}
	desired, err := GenerateSigningKeySecret(cr)
	if err != nil {
		return err
	}
	if err := controllerutilSetOwnerRef(cr, desired, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, desired); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create signing Secret: %w", err)
	}
	return nil
}

// reconcileQdrant creates / updates / deletes the in-namespace Qdrant
// StatefulSet + ClusterIP Service that back the RAG memory loop when
// spec.ai.memory.enabled=true. Mirrors the chart's
// templates/rag-qdrant-*.yaml.
//
// IMPORTANT: K8s rejects mutations to StatefulSet.spec.{selector,
// serviceName,volumeClaimTemplates,podManagementPolicy} after create
// — the only safe in-place mutations are spec.{replicas,template,
// updateStrategy,minReadySeconds}. So the Update path explicitly
// overwrites just those.
func (r *Reconciler) reconcileQdrant(ctx context.Context, cr *chav1alpha1.ClusterHealthAutopilot) error {
	ns := cr.Namespace
	name := NamesFor(cr).RAG

	desiredSS := BuildQdrantStatefulSet(cr)
	desiredSvc := BuildQdrantService(cr)

	if desiredSS == nil {
		// Memory off — tear down whatever we already manage. The PVC
		// is intentionally NOT deleted (StatefulSet's standard
		// behavior; operators who want PVC cleanup do it explicitly).
		if err := r.deleteIfExists(ctx, &appsv1.StatefulSet{}, ns, name); err != nil {
			return fmt.Errorf("delete qdrant StatefulSet: %w", err)
		}
		return r.deleteIfExists(ctx, &corev1.Service{}, ns, name)
	}

	if err := controllerutilSetOwnerRef(cr, desiredSS, r.Scheme); err != nil {
		return err
	}
	if err := controllerutilSetOwnerRef(cr, desiredSvc, r.Scheme); err != nil {
		return err
	}

	// Service first — the StatefulSet's ServiceName references it.
	if err := r.createOrUpdate(ctx, desiredSvc, func(current client.Object) {
		c := current.(*corev1.Service)
		// Don't touch ClusterIP (immutable) or live Endpoints. Only
		// spec.ports + spec.selector + labels need to track the
		// desired state.
		c.Spec.Ports = desiredSvc.Spec.Ports
		c.Spec.Selector = desiredSvc.Spec.Selector
		mergeLabels(&c.ObjectMeta, desiredSvc.Labels)
	}); err != nil {
		return fmt.Errorf("qdrant Service: %w", err)
	}
	return r.createOrUpdate(ctx, desiredSS, func(current client.Object) {
		c := current.(*appsv1.StatefulSet)
		// Only mutate the mutable fields. Leave Selector / ServiceName /
		// VolumeClaimTemplates alone — those land at first Create and
		// the apiserver rejects any change after.
		c.Spec.Replicas = desiredSS.Spec.Replicas
		c.Spec.Template = desiredSS.Spec.Template
		c.Spec.UpdateStrategy = desiredSS.Spec.UpdateStrategy
		mergeLabels(&c.ObjectMeta, desiredSS.Labels)
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

	// ReaderRBACReady — True iff BOTH the shared ClusterRole and the
	// per-CR ClusterRoleBinding exist + the binding targets the CR's
	// SA. Either missing → False (the watcher would get `forbidden`
	// on every List). Get-error → Unknown.
	var rbacStatus metav1.ConditionStatus
	var rbacReason, rbacMsg string
	var role rbacv1.ClusterRole
	roleErr := r.Get(ctx, types.NamespacedName{Name: ReaderClusterRoleName}, &role)
	var binding rbacv1.ClusterRoleBinding
	bindingErr := r.Get(ctx, types.NamespacedName{Name: ReaderClusterRoleBindingName(cr)}, &binding)
	switch {
	case apierrors.IsNotFound(roleErr):
		rbacStatus = metav1.ConditionFalse
		rbacReason = "ClusterRoleMissing"
		rbacMsg = "reader ClusterRole " + ReaderClusterRoleName + " not found"
	case apierrors.IsNotFound(bindingErr):
		rbacStatus = metav1.ConditionFalse
		rbacReason = "ClusterRoleBindingMissing"
		rbacMsg = "reader ClusterRoleBinding " + ReaderClusterRoleBindingName(cr) + " not found"
	case roleErr != nil:
		rbacStatus = metav1.ConditionUnknown
		rbacReason = "GetFailed"
		rbacMsg = roleErr.Error()
	case bindingErr != nil:
		rbacStatus = metav1.ConditionUnknown
		rbacReason = "GetFailed"
		rbacMsg = bindingErr.Error()
	case !bindingTargetsSA(&binding, ServiceAccountNameFor(cr), cr.Namespace):
		rbacStatus = metav1.ConditionFalse
		rbacReason = "WrongSubject"
		rbacMsg = "binding does not target the CR's ServiceAccount"
	default:
		rbacStatus = metav1.ConditionTrue
		rbacReason = "Provisioned"
		rbacMsg = "reader ClusterRole + per-CR binding present"
	}
	setCondition(&cr.Status, chav1alpha1.ConditionReaderRBACReady,
		rbacStatus, rbacReason, rbacMsg, cr.Generation)

	// AIWatchRunning (Phase 2) — same shape as WatcherRunning. Set
	// even when AI is disabled so dashboards can show "AI off"
	// explicitly rather than inferring from a missing condition.
	var aiStatus metav1.ConditionStatus
	var aiReason, aiMsg string
	if !AIEnabled(cr) {
		aiStatus = metav1.ConditionFalse
		aiReason = "Disabled"
		aiMsg = "spec.ai.enabled is false"
	} else {
		var dep appsv1.Deployment
		err := r.Get(ctx, types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      NamesFor(cr).AIWatch,
		}, &dep)
		switch {
		case apierrors.IsNotFound(err):
			aiStatus = metav1.ConditionFalse
			aiReason = "DeploymentMissing"
			aiMsg = "aiwatch Deployment not found"
		case err != nil:
			aiStatus = metav1.ConditionUnknown
			aiReason = "GetFailed"
			aiMsg = err.Error()
		case dep.Status.AvailableReplicas > 0:
			aiStatus = metav1.ConditionTrue
			aiReason = "AvailableReplicasPositive"
			aiMsg = fmt.Sprintf("availableReplicas=%d/%d",
				dep.Status.AvailableReplicas, *dep.Spec.Replicas)
		default:
			aiStatus = metav1.ConditionFalse
			aiReason = "NoAvailableReplicas"
			aiMsg = "aiwatch Deployment has no available replicas"
		}
	}
	setCondition(&cr.Status, chav1alpha1.ConditionAIWatchRunning,
		aiStatus, aiReason, aiMsg, cr.Generation)

	// MemoryStoreReady (Phase 2b) — True only when BOTH the Qdrant
	// Service and StatefulSet exist AND the StatefulSet reports
	// readyReplicas > 0. Either missing → False; only Disabled when
	// memory is off in spec.
	var memStatus metav1.ConditionStatus
	var memReason, memMsg string
	if !MemoryEnabled(cr) {
		memStatus = metav1.ConditionFalse
		memReason = "Disabled"
		memMsg = "spec.ai.memory.enabled is false"
	} else {
		var svc corev1.Service
		svcErr := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: NamesFor(cr).RAG}, &svc)
		var ss appsv1.StatefulSet
		ssErr := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: NamesFor(cr).RAG}, &ss)
		switch {
		case apierrors.IsNotFound(svcErr):
			memStatus = metav1.ConditionFalse
			memReason = "ServiceMissing"
			memMsg = "Qdrant Service " + NamesFor(cr).RAG + " not found"
		case apierrors.IsNotFound(ssErr):
			memStatus = metav1.ConditionFalse
			memReason = "StatefulSetMissing"
			memMsg = "Qdrant StatefulSet " + NamesFor(cr).RAG + " not found"
		case svcErr != nil:
			memStatus = metav1.ConditionUnknown
			memReason = "GetFailed"
			memMsg = svcErr.Error()
		case ssErr != nil:
			memStatus = metav1.ConditionUnknown
			memReason = "GetFailed"
			memMsg = ssErr.Error()
		case ss.Status.ReadyReplicas > 0:
			memStatus = metav1.ConditionTrue
			memReason = "ReadyReplicasPositive"
			memMsg = fmt.Sprintf("readyReplicas=%d/%d",
				ss.Status.ReadyReplicas, *ss.Spec.Replicas)
		default:
			memStatus = metav1.ConditionFalse
			memReason = "NoReadyReplicas"
			memMsg = "Qdrant StatefulSet has no ready replicas"
		}
	}
	setCondition(&cr.Status, chav1alpha1.ConditionMemoryStoreReady,
		memStatus, memReason, memMsg, cr.Generation)

	// ApprovalServerReady (Phase 2c-B) — True only when signing-key
	// Secret has both keys AND Deployment reports availableReplicas > 0
	// AND Service exists. Disabled when approval is off.
	var apStatus metav1.ConditionStatus
	var apReason, apMsg string
	if !ApprovalEnabled(cr) {
		apStatus = metav1.ConditionFalse
		apReason = "Disabled"
		apMsg = "spec.approval.enabled is false"
	} else {
		var secret corev1.Secret
		secErr := r.Get(ctx, types.NamespacedName{
			Namespace: cr.Namespace, Name: ApprovalSigningKeySecretName(cr),
		}, &secret)
		var svc corev1.Service
		svcErr := r.Get(ctx, types.NamespacedName{
			Namespace: cr.Namespace, Name: ApprovalServerName(cr),
		}, &svc)
		var dep appsv1.Deployment
		depErr := r.Get(ctx, types.NamespacedName{
			Namespace: cr.Namespace, Name: ApprovalServerName(cr),
		}, &dep)
		switch {
		case apierrors.IsNotFound(secErr):
			apStatus = metav1.ConditionFalse
			apReason = "SigningKeyMissing"
			apMsg = "signing-key Secret " + ApprovalSigningKeySecretName(cr) + " not found"
		case secErr != nil:
			apStatus = metav1.ConditionUnknown
			apReason = "GetFailed"
			apMsg = secErr.Error()
		case len(secret.Data["signing.key"]) == 0 || len(secret.Data["signing.pub"]) == 0:
			apStatus = metav1.ConditionFalse
			apReason = "SigningKeyIncomplete"
			apMsg = "signing-key Secret missing signing.key or signing.pub"
		case apierrors.IsNotFound(svcErr):
			apStatus = metav1.ConditionFalse
			apReason = "ServiceMissing"
			apMsg = "approval-server Service " + ApprovalServerName(cr) + " not found"
		case apierrors.IsNotFound(depErr):
			apStatus = metav1.ConditionFalse
			apReason = "DeploymentMissing"
			apMsg = "approval-server Deployment " + ApprovalServerName(cr) + " not found"
		case svcErr != nil || depErr != nil:
			apStatus = metav1.ConditionUnknown
			apReason = "GetFailed"
			if svcErr != nil {
				apMsg = svcErr.Error()
			} else {
				apMsg = depErr.Error()
			}
		case dep.Status.AvailableReplicas > 0:
			apStatus = metav1.ConditionTrue
			apReason = "AvailableReplicasPositive"
			apMsg = fmt.Sprintf("availableReplicas=%d/%d",
				dep.Status.AvailableReplicas, *dep.Spec.Replicas)
		default:
			apStatus = metav1.ConditionFalse
			apReason = "NoAvailableReplicas"
			apMsg = "approval-server Deployment has no available replicas"
		}
	}
	setCondition(&cr.Status, chav1alpha1.ConditionApprovalServerReady,
		apStatus, apReason, apMsg, cr.Generation)

	// Ready — True iff reconcile had no error AND every other
	// condition that gates correctness is True. Each AI/memory/
	// approval condition gates Ready ONLY when its feature is
	// enabled, so installs that don't opt in stay Ready=True.
	readyStatus := metav1.ConditionTrue
	readyReason := "Reconciled"
	readyMsg := "all subresources reconciled"
	switch {
	case firstErr != nil:
		readyStatus = metav1.ConditionFalse
		readyReason = "ReconcileError"
		readyMsg = firstErr.Error()
	case rbacStatus != metav1.ConditionTrue:
		readyStatus = metav1.ConditionFalse
		readyReason = "ReaderRBACNotReady"
		readyMsg = "ReaderRBACReady=" + string(rbacStatus) + ": " + rbacMsg
	case AIEnabled(cr) && aiStatus != metav1.ConditionTrue:
		readyStatus = metav1.ConditionFalse
		readyReason = "AIWatchNotReady"
		readyMsg = "AIWatchRunning=" + string(aiStatus) + ": " + aiMsg
	case MemoryEnabled(cr) && memStatus != metav1.ConditionTrue:
		readyStatus = metav1.ConditionFalse
		readyReason = "MemoryStoreNotReady"
		readyMsg = "MemoryStoreReady=" + string(memStatus) + ": " + memMsg
	case ApprovalEnabled(cr) && apStatus != metav1.ConditionTrue:
		readyStatus = metav1.ConditionFalse
		readyReason = "ApprovalServerNotReady"
		readyMsg = "ApprovalServerReady=" + string(apStatus) + ": " + apMsg
	}
	setCondition(&cr.Status, chav1alpha1.ConditionReady,
		readyStatus, readyReason, readyMsg, cr.Generation)

	cr.Status.ObservedGeneration = cr.Generation
}

// bindingTargetsSA reports whether a ClusterRoleBinding lists the
// given ServiceAccount among its Subjects. Used by the condition
// computer to detect a stale binding whose Subject no longer matches
// the CR's SA (e.g. operator updated spec.serviceAccountName).
func bindingTargetsSA(b *rbacv1.ClusterRoleBinding, saName, saNamespace string) bool {
	for _, s := range b.Subjects {
		if s.Kind == rbacv1.ServiceAccountKind &&
			s.Name == saName &&
			s.Namespace == saNamespace {
			return true
		}
	}
	return false
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
// manager. Owns() the child resource types so changes to them requeue
// the parent CR. Phase 2b adds StatefulSet + Service (Qdrant). Phase
// 2c-B adds Secret + Role + RoleBinding (approval-server). Phase 2c-C
// adds Ingress (approval-server, optional).
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chav1alpha1.ClusterHealthAutopilot{}, builder.WithPredicates()).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&batchv1.CronJob{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&networkingv1.Ingress{}).
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

// labelsUpToDate returns true iff every key=value in desired is
// already present in current. Extra keys in current are ignored —
// users may add their own labels without them being wiped.
func labelsUpToDate(current, desired map[string]string) bool {
	for k, v := range desired {
		if current[k] != v {
			return false
		}
	}
	return true
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
