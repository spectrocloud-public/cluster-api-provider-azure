/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package asomigration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

// storedVersionMigration describes an ASO-managed CRD whose
// status.storedVersions still references a version that the bundled ASO
// release no longer lists in spec.versions. The Kubernetes API server rejects
// a CRD update that drops such a version until a storage migration ensures no
// data remains persisted in it, so an upgrade would otherwise fail with:
//
//	status.storedVersions[i]: Invalid value: "<version>": missing from
//	spec.versions; ... must remain in spec.versions until a storage migration
//	ensures no data remains persisted in <version> ...
type storedVersionMigration struct {
	// crd is the name of the affected CustomResourceDefinition.
	crd string
	// stale is the storedVersions entry the bundled ASO no longer serves.
	stale string
}

// storedVersionMigrations lists the known-affected ASO-managed CRDs. Add an
// entry here when a future ASO release removes a previously-stored version.
// ASO v2.18 removes containerservice/v1api20231001 from ManagedCluster and
// AgentPool; clusters that ran an early CAPZ-bundled ASO (v2.5-v2.8, where
// v1api20231001storage was the hub) may still list it in status.storedVersions.
var storedVersionMigrations = []storedVersionMigration{
	{crd: "fleetsmembers.containerservice.azure.com", stale: "v1api20230315previewstorage"},
	{crd: "managedclusters.containerservice.azure.com", stale: "v1api20231001storage"},
	{crd: "managedclustersagentpools.containerservice.azure.com", stale: "v1api20231001storage"},
}

// asoFinalizer is the finalizer ASO stamps on every custom resource it
// manages. A delete of a resource carrying it only sets deletionTimestamp
// until ASO processes the finalizer, which (on the Azure side) would delete
// the real resource. This migration removes it before deleting instances.
const asoFinalizer = "serviceoperator.azure.com/finalizer"

// capzOwnerGroup is the API group every CAPZ custom resource lives in. An ASO
// instance with an ownerReference into this group was created by CAPZ's
// managed-cluster controllers as a mirror of the authoritative AzureManaged*
// resources, and is recreated by CAPZ's adopt controllers after the CRD is
// replaced. Instances without such an owner were created by something else
// and are never touched by this migration.
const capzOwnerGroup = "infrastructure.cluster.x-k8s.io"

// crdDeletionTimeout bounds how long we wait for a deleted CRD to disappear.
const crdDeletionTimeout = 2 * time.Minute

// instanceDeletionTimeout bounds how long we wait for deleted instances to
// disappear before the CRD itself is deleted.
const instanceDeletionTimeout = 30 * time.Second

// MigrateStoredVersions cleans up obsolete status.storedVersions entries for
// ASO-managed CRDs before the bundled ASO controller tries to apply them. It
// runs as an init container in the ASO deployment (via the `manager
// migrate-aso-crds` subcommand) so it completes before ASO reconciles its CRDs,
// reusing the CAPZ manager image instead of depending on an external one.
//
// A CRD is only deleted once its emptiness has been confirmed through reads
// that cannot involve the ASO conversion webhook (see migrateStoredVersion) and
// once ASO's leader-election lease proves no ASO controller is running (see
// confirmASOControllersNotRunning). Instances created by CAPZ's managed-cluster
// controllers are cleaned up automatically; anything else fails closed with
// instructions for a human.
func MigrateStoredVersions(ctx context.Context, cfg *rest.Config) error {
	log := ctrl.Log.WithName("migrate-aso-crds")

	crdClient, err := apiextensionsclientset.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating apiextensions client: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating dynamic client: %w", err)
	}

	for _, m := range storedVersionMigrations {
		if err := migrateStoredVersion(ctx, log, crdClient, dynClient, m); err != nil {
			return err
		}
	}
	return nil
}

func migrateStoredVersion(ctx context.Context, log logr.Logger, crdClient apiextensionsclientset.Interface, dynClient dynamic.Interface, m storedVersionMigration) error {
	crd, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, m.crd, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		log.Info("CRD not present; skipping", "crd", m.crd)
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting CRD %s: %w", m.crd, err)
	}

	if !slices.Contains(crd.Status.StoredVersions, m.stale) {
		log.Info("CRD has no stale stored version; skipping", "crd", m.crd, "staleVersion", m.stale)
		return nil
	}

	// The ASO conversion webhook lives in the ASO pod whose init container is
	// running this migration, so it can never be reachable here: the main
	// container does not start until init completes. The API server routes a
	// custom-resource read through that webhook whenever the requested version
	// differs from the encoding an object is persisted in, so which lists are
	// safe depends on status.storedVersions:
	//
	//   - When the stale version is the ONLY stored version it is necessarily
	//     the storage version (the API server always keeps the storage version
	//     listed), so every persisted object is in that single encoding. A
	//     list at it decodes the etcd-native form and converts nothing: the
	//     count is exact and conversion-free. This is the state of a cluster
	//     whose bundled ASO has always stored at the now-stale version, which
	//     is the common upgrade shape.
	//   - With multiple stored versions, objects may exist in more than one
	//     encoding and no single list is both complete and conversion-free.
	//     Emptiness is still verifiable, because an empty result never reaches
	//     the webhook (the API server short-circuits conversion when no object
	//     needs converting): enumerate the served versions and only a provably
	//     empty CRD may be deleted.
	if gvr, ok := storageVersionGVR(crd, m.stale); ok {
		return migrateSoleStorageVersion(ctx, log, crdClient, dynClient, m, crd, gvr)
	}
	return migrateMultiStoredVersions(ctx, log, crdClient, dynClient, m, crd)
}

// migrateSoleStorageVersion handles a CRD whose stale version is the only
// stored version and is served: listing at it sees every persisted object
// without conversion. An empty CRD is deleted; a CRD holding only instances
// created by CAPZ's controllers is cleaned up; anything else fails closed.
func migrateSoleStorageVersion(ctx context.Context, log logr.Logger, crdClient apiextensionsclientset.Interface, dynClient dynamic.Interface, m storedVersionMigration, crd *apiextensionsv1.CustomResourceDefinition, gvr schema.GroupVersionResource) error {
	objs, err := listInstances(ctx, dynClient, gvr)
	if err != nil {
		// Fail closed: a real listing failure (API server hiccup, RBAC denial)
		// must never be misread as "zero instances" and let us delete a CRD
		// that still holds data.
		return fmt.Errorf("listing %s to confirm CRD %s is empty: %w", gvr, m.crd, err)
	}

	if len(objs) == 0 {
		return deleteEmptyCRD(ctx, log, crdClient, m.crd, crd.Status.StoredVersions)
	}

	foreign := 0
	for i := range objs {
		if !isCAPZOwned(&objs[i]) {
			foreign++
		}
	}
	if foreign > 0 {
		return fmt.Errorf("CRD %s holds %d instance(s) in stale storage version %q, %d of which "+
			"are not owned by a cluster-api-provider-azure resource. This migration only cleans up "+
			"instances created by CAPZ's managed-cluster controllers. Delete or migrate the foreign "+
			"instances manually (or run 'asoctl clean crds' from a workstation while the currently "+
			"installed ASO release's conversion webhook is reachable), then retry the CAPZ upgrade",
			m.crd, len(objs), m.stale, foreign)
	}

	// Destructive from here on, so first prove that no ASO controller is
	// running: a live one could re-add the finalizer between our patch and our
	// delete and then process the delete against the real Azure resource (see
	// confirmASOControllersNotRunning).
	if err := confirmASOControllersNotRunning(ctx, dynClient, gvr); err != nil {
		return err
	}

	log.Info("CRD holds only CAPZ-owned instances; removing their finalizers and deleting them so the new ASO controller can recreate the CRD",
		"crd", m.crd, "instances", len(objs), "storedVersions", crd.Status.StoredVersions)
	for i := range objs {
		obj := &objs[i]
		if err := deleteOwnedOperatorSpecOutputs(ctx, log, dynClient, obj); err != nil {
			return err
		}
		if err := stripASOFinalizer(ctx, dynClient, gvr, obj); err != nil {
			return fmt.Errorf("removing %s from %s %s/%s: %w", asoFinalizer, gvr.Resource, obj.GetNamespace(), obj.GetName(), err)
		}
		if err := dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("deleting %s %s/%s: %w", gvr.Resource, obj.GetNamespace(), obj.GetName(), err)
		}
		log.Info("Deleted instance", "resource", gvr.Resource, "namespace", obj.GetNamespace(), "name", obj.GetName())
	}
	if err := waitForInstancesGone(ctx, dynClient, gvr); err != nil {
		return fmt.Errorf("waiting for instances of %s to be deleted: %w", m.crd, err)
	}

	log.Info("CRD drained and stale storedVersions; deleting so the new ASO controller can recreate it",
		"crd", m.crd, "instancesDeleted", len(objs))
	return deleteCRD(ctx, log, crdClient, m.crd)
}

// migrateMultiStoredVersions handles a CRD whose stale version is one of
// several stored versions (or is unserved). Objects may exist in more than one
// encoding, so no single list is both complete and conversion-free; the
// migration verifies emptiness across the served versions (a webhook-free
// read whenever the CRD is empty) and deletes the CRD only then. Any data or
// any listing failure fails closed.
func migrateMultiStoredVersions(ctx context.Context, log logr.Logger, crdClient apiextensionsclientset.Interface, dynClient dynamic.Interface, m storedVersionMigration, crd *apiextensionsv1.CustomResourceDefinition) error {
	// Every served version exposes the same stored objects (the API server
	// converts on the fly), so take the max instance count across the served
	// versions rather than summing, which would overcount by the number of
	// versions. Unserved versions (e.g. the stale storage version) can't be
	// listed directly, but their stored objects surface through the served
	// versions we do list.
	instances := 0
	for _, v := range crd.Spec.Versions {
		if !v.Served {
			continue
		}
		gvr := schema.GroupVersionResource{
			Group:    crd.Spec.Group,
			Version:  v.Name,
			Resource: crd.Spec.Names.Plural,
		}
		n, err := countInstances(ctx, dynClient, gvr)
		if err != nil {
			// Fail closed: a real listing failure (conversion webhook down,
			// API server hiccup, RBAC denial) must never be misread as "zero
			// instances" and let us delete a CRD that still holds data.
			return fmt.Errorf("listing %s to confirm CRD %s is empty: %w", gvr, m.crd, err)
		}
		instances = max(instances, n)
	}

	if instances != 0 {
		return fmt.Errorf("CRD %s has %d instance(s) but its status.storedVersions still references %q, "+
			"which the bundled Azure Service Operator no longer serves. These instances may exist in more than one "+
			"stored encoding and cannot be verified safely from inside the ASO pod. Run 'asoctl clean crds' from a "+
			"workstation while an ASO pod is serving, or, if no ASO pod is running and the conversion webhook is "+
			"unreachable, delete the remaining instances manually, before retrying the CAPZ upgrade. See "+
			"https://azure.github.io/azure-service-operator/guide/crd-management/ for details",
			m.crd, instances, m.stale)
	}

	return deleteEmptyCRD(ctx, log, crdClient, m.crd, crd.Status.StoredVersions)
}

// isCAPZOwned reports whether the object carries an ownerReference into the
// CAPZ API group. Every ASO instance CAPZ's managed-cluster controllers create
// (the mirrors backing AzureManagedControlPlane and AzureManagedMachinePool)
// carries such a reference, and CAPZ's adopt controllers recreate those
// mirrors from the authoritative resources after the CRD is replaced.
// Instances without such an owner were created by something else (a human or
// another operator) and must never be deleted by an automated init container.
func isCAPZOwned(obj *unstructured.Unstructured) bool {
	for _, ref := range obj.GetOwnerReferences() {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			continue
		}
		if gv.Group == capzOwnerGroup {
			return true
		}
	}
	return false
}

// operatorSpecOutputGVRs maps the two kinds an ASO CR's spec.operatorSpec can
// name as write destinations to their GVRs. ASO's "additional Kubernetes
// objects" phase (genruntime.ApplyObjsAndEnsureOwner) is the only place it
// stamps ownerReferences onto objects it did not create from CRs, and its
// exporters emit exactly Secrets and ConfigMaps there, so these two resources
// are the complete set of possible dependents.
var operatorSpecOutputGVRs = map[string]schema.GroupVersionResource{
	"Secret":    {Group: "", Version: "v1", Resource: "secrets"},
	"ConfigMap": {Group: "", Version: "v1", Resource: "configmaps"},
}

// operatorSpecDependency is one Secret or ConfigMap that a CR's
// spec.operatorSpec asks ASO to write in the CR's namespace. ASO stamps each
// such object with an ownerReference to the CR.
type operatorSpecDependency struct {
	name string
	kind string // "Secret" or "ConfigMap"
}

// operatorSpecDependencies returns the Secret and ConfigMap destinations
// declared in the CR's spec.operatorSpec, secrets before configmaps and each
// group in sorted order for deterministic behavior.
func operatorSpecDependencies(obj *unstructured.Unstructured) []operatorSpecDependency {
	var deps []operatorSpecDependency
	for _, section := range []struct{ kind, field string }{
		{kind: "Secret", field: "secrets"},
		{kind: "ConfigMap", field: "configMaps"},
	} {
		entries, found, _ := unstructured.NestedMap(obj.Object, "spec", "operatorSpec", section.field)
		if !found {
			continue
		}
		names := make([]string, 0, len(entries))
		for name := range entries {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			dest, ok := entries[name].(map[string]any)
			if !ok {
				continue
			}
			target, _, _ := unstructured.NestedString(dest, "name")
			if target == "" {
				continue
			}
			deps = append(deps, operatorSpecDependency{name: target, kind: section.kind})
		}
	}
	return deps
}

// deleteOwnedOperatorSpecOutputs deletes every Secret or ConfigMap declared in
// obj's spec.operatorSpec that carries an ownerReference to obj (whatever the
// reference's UID). It runs while draining instances whose CRD is about to be
// deleted, because deletion is the only way such a dependent can ever be
// reconciled again: the CR's UID dies with it, so the recreated CR cannot own
// the old object, the garbage collector cannot reclaim it (its ownerReference
// names an apiVersion the recreated CRD no longer serves, so the owner never
// resolves), and ASO refuses to write an existing object it does not own by
// UID (genruntime.CheckTargetOwnedByObj treats an object with no matching
// reference exactly like a foreign-owned one, on the grounds that it may be
// user-created). Deleting the stranded output lets ASO create a fresh one
// owned by the recreated CR on its next reconcile. Outputs without a reference
// naming obj are left in place: they may be user-owned, and this migration has
// no more business deleting those than ASO has overwriting them.
func deleteOwnedOperatorSpecOutputs(ctx context.Context, log logr.Logger, dynClient dynamic.Interface, obj *unstructured.Unstructured) error {
	group, err := schema.ParseGroupVersion(obj.GetAPIVersion())
	if err != nil {
		return fmt.Errorf("parsing the apiVersion of %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
	}
	for _, dep := range operatorSpecDependencies(obj) {
		gvr, ok := operatorSpecOutputGVRs[dep.kind]
		if !ok {
			continue
		}
		output, err := dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Get(ctx, dep.name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("getting %s %s/%s declared in the operatorSpec of %s %s/%s: %w",
				dep.kind, obj.GetNamespace(), dep.name, obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
		}
		refUID, found := ownerRefToCR(output, group.Group, obj.GetKind(), obj.GetName())
		if !found {
			continue
		}
		if err := dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Delete(ctx, dep.name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("deleting the operatorSpec output of %s %s/%s being deleted: %s %s/%s (owner UID %s): %w",
				obj.GetKind(), obj.GetName(), obj.GetNamespace(), dep.kind, obj.GetNamespace(), dep.name, refUID, err)
		}
		log.Info("Deleted the operatorSpec output of the CR being deleted",
			"kind", dep.kind, "namespace", obj.GetNamespace(), "name", dep.name,
			"cr", obj.GetName(), "ownerUID", refUID)
	}
	return nil
}

// ownerRefToCR returns the UID of the first ownerReference on target naming
// the given group, kind and name, and whether such a reference exists.
func ownerRefToCR(target *unstructured.Unstructured, group, kind, name string) (refUID types.UID, found bool) {
	for _, ref := range target.GetOwnerReferences() {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			continue
		}
		if gv.Group != group || ref.Kind != kind || ref.Name != name {
			continue
		}
		return ref.UID, true
	}
	return "", false
}

// currentStorageGVR returns the GVR of the CRD's storage version when it is
// the only entry in status.storedVersions and is served: the only state in
// which listing at it is a complete, conversion-free view of every persisted
// object (see storageVersionGVR). It returns false in any other state.
func currentStorageGVR(crd *apiextensionsv1.CustomResourceDefinition) (schema.GroupVersionResource, bool) {
	if len(crd.Status.StoredVersions) != 1 {
		return schema.GroupVersionResource{}, false
	}
	for _, v := range crd.Spec.Versions {
		if v.Name == crd.Status.StoredVersions[0] && v.Served && v.Storage {
			return schema.GroupVersionResource{
				Group:    crd.Spec.Group,
				Version:  v.Name,
				Resource: crd.Spec.Names.Plural,
			}, true
		}
	}
	return schema.GroupVersionResource{}, false
}

// storageVersionGVR returns the GVR of the stale version when it is the only
// entry in status.storedVersions and is both served and the storage version
// (the only state in which listing at it is a complete, conversion-free view
// of every persisted object). It returns false in any other state.
func storageVersionGVR(crd *apiextensionsv1.CustomResourceDefinition, stale string) (schema.GroupVersionResource, bool) {
	if len(crd.Status.StoredVersions) != 1 || crd.Status.StoredVersions[0] != stale {
		return schema.GroupVersionResource{}, false
	}
	return currentStorageGVR(crd)
}

// asoLeaderElectionLease is the Lease controller-runtime's leader election
// uses for the ASO manager. ASO hardcodes it as its LeaderElectionID in
// cmd/controller/app/setup.go, so every ASO v2 manager acquires exactly this
// lease in the ASO deployment's namespace before reconciling anything.
const asoLeaderElectionLease = "controllers-leader-election-azinfra-generated"

// asoLeaseDurationFallback is the lease duration assumed when the Lease object
// does not carry one. controller-runtime's manager defaults to 15s and the
// bundled ASO does not override it, so this matches the observed leases.
const asoLeaseDurationFallback = 15 * time.Second

// leaseRenewalClockSkewMargin is the extra staleness allowed beyond the lease
// duration. A renewal timestamp is written with the holding manager's node
// clock and compared against this pod's clock, so a margin well beyond
// realistic node skew keeps a live manager from being mistaken for a dead one.
// Overestimating only delays the migration by one init retry (fail closed);
// underestimating could race a live ASO manager and delete Azure resources.
const leaseRenewalClockSkewMargin = time.Minute

// serviceAccountNamespaceFile is the projected service account namespace file
// present in every pod. The migration runs as an init container in the ASO
// pod, so it names the ASO deployment's namespace, which is where the ASO
// manager acquires its leader-election lease. A variable so tests can point it
// at a fixture.
var serviceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// podNamespace returns the namespace of the pod this migration runs in.
func podNamespace() (string, error) {
	b, err := os.ReadFile(serviceAccountNamespaceFile)
	if err != nil {
		return "", err
	}
	ns := strings.TrimSpace(string(b))
	if ns == "" {
		return "", fmt.Errorf("%s is empty", serviceAccountNamespaceFile)
	}
	return ns, nil
}

// confirmASOControllersNotRunning verifies that no ASO manager holds leadership
// before the caller mutates cluster state. Live ASO controllers make a delete
// dangerous: ASO can re-add the finalizer between our patch and our delete,
// turning the delete into a normal ASO-processed deletion that removes the
// real Azure resource.
//
// The check reads ASO's leader-election Lease rather than probing the
// conversion webhook. A previous revision listed the CRD at a non-storage
// version and treated a successful response as proof that a serving ASO pod
// was converting stored objects, but this cluster's API server serves
// data-bearing lists at non-storage versions conversion-free even when the
// webhook is provably down ("no endpoints available", observed across several
// ASO CRDs in two API groups), so success proved nothing and the migration
// wedged. The lease is the signal leader election itself relies on: the ASO
// manager acquires it and renews it every few seconds while running, and
// controller-runtime releases it on graceful shutdown. A lease that is absent,
// released (empty holder), or not renewed for its full duration proves no
// manager is running. The bundled ASO deployment runs with
// --enable-leader-election and a single replica under the Recreate strategy,
// so "no lease activity" cannot coexist with a live manager.
//
// The ASO service account already carries leases get/list for leader election
// in its own namespace, so the check needs no new RBAC. Anything unexpected (a
// read error, a holder without a parsable renewal time) fails closed: the init
// container retries, and a human investigates the message.
func confirmASOControllersNotRunning(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	ns, err := podNamespace()
	if err != nil {
		return fmt.Errorf("determining the namespace of this ASO pod to read its leader-election lease: %w", err)
	}
	leases := dynClient.Resource(schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}).Namespace(ns)
	lease, err := leases.Get(ctx, asoLeaderElectionLease, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting the ASO leader-election lease %s/%s: %w", ns, asoLeaderElectionLease, err)
	}

	holder, _, _ := unstructured.NestedString(lease.Object, "spec", "holderIdentity")
	if holder == "" {
		// controller-runtime releases the lease on graceful shutdown by
		// clearing the holder, so an empty holder is a manager that exited.
		return nil
	}
	renewTimeStr, found, _ := unstructured.NestedString(lease.Object, "spec", "renewTime")
	if !found {
		return fmt.Errorf("the ASO leader-election lease %s/%s names holder %q but carries no renewTime, "+
			"so its freshness cannot be established; refusing to modify %s", ns, asoLeaderElectionLease, holder, gvr.Resource)
	}
	renewTime, err := time.Parse(time.RFC3339, renewTimeStr)
	if err != nil {
		return fmt.Errorf("parsing the renewTime %q of the ASO leader-election lease %s/%s: %w",
			renewTimeStr, ns, asoLeaderElectionLease, err)
	}

	duration := asoLeaseDurationFallback
	if raw, found, _ := unstructured.NestedFieldNoCopy(lease.Object, "spec", "leaseDurationSeconds"); found {
		switch n := raw.(type) {
		case float64:
			duration = time.Duration(n) * time.Second
		case json.Number:
			if f, err := n.Float64(); err == nil {
				duration = time.Duration(f) * time.Second
			}
		}
	}

	staleness := time.Since(renewTime)
	if staleness > duration+leaseRenewalClockSkewMargin {
		return nil
	}
	return fmt.Errorf("the Azure Service Operator controllers are still running: the leader-election lease %s/%s "+
		"is held by %q and was renewed %s ago, which is inside its %s staleness threshold. Deleting instances of %s "+
		"now could race with the running ASO controllers: ASO can re-add the finalizer between this migration's patch "+
		"and delete, then process the deletion and remove the real Azure resource. Wait until no azure-service-operator "+
		"pod is running, then retry",
		ns, asoLeaderElectionLease, holder, staleness.Round(time.Second), duration+leaseRenewalClockSkewMargin, gvr.Resource)
}

// stripASOFinalizer removes the ASO finalizer from obj via a merge patch,
// leaving any other finalizers in place. The patch (unlike an update) carries
// no resourceVersion, so it cannot conflict with a concurrent writer; a
// not-found error is tolerated because the object may have been deleted since
// it was listed.
func stripASOFinalizer(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) error {
	finalizers := obj.GetFinalizers()
	if !slices.Contains(finalizers, asoFinalizer) {
		return nil
	}
	remaining := slices.DeleteFunc(slices.Clone(finalizers), func(f string) bool { return f == asoFinalizer })
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{"finalizers": remaining}})
	if err != nil {
		return err
	}
	_, err = dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Patch(ctx, obj.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// deleteEmptyCRD logs and deletes a CRD whose emptiness has been confirmed.
func deleteEmptyCRD(ctx context.Context, log logr.Logger, crdClient apiextensionsclientset.Interface, name string, storedVersions []string) error {
	log.Info("CRD has no instances and stale storedVersions; deleting so the new ASO controller can recreate it",
		"crd", name, "storedVersions", storedVersions)
	return deleteCRD(ctx, log, crdClient, name)
}

// deleteCRD deletes the named CRD and waits for it to disappear, tolerating a
// concurrent deletion. The new ASO controller recreates it from its bundled
// definition with a status.storedVersions containing only the new storage
// version.
func deleteCRD(ctx context.Context, log logr.Logger, crdClient apiextensionsclientset.Interface, name string) error {
	if err := crdClient.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting CRD %s: %w", name, err)
	}
	if err := waitForCRDDeletion(ctx, crdClient, name); err != nil {
		return err
	}
	log.Info("CRD deleted", "crd", name)
	return nil
}

// listInstances returns every object of the given resource across all
// namespaces, following continuation tokens. Any list error is returned to the
// caller, which fails closed rather than risk deleting a CRD whose contents it
// couldn't inspect.
func listInstances(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource) ([]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured
	opts := metav1.ListOptions{Limit: 500}
	for {
		list, err := dynClient.Resource(gvr).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, list.Items...)
		cont := list.GetContinue()
		if cont == "" {
			return out, nil
		}
		opts.Continue = cont
	}
}

// countInstances returns the number of objects of the given resource across
// all namespaces.
func countInstances(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource) (int, error) {
	objs, err := listInstances(ctx, dynClient, gvr)
	if err != nil {
		return 0, err
	}
	return len(objs), nil
}

// waitForInstancesGone blocks until the given resource has no instances left
// or the timeout elapses.
func waitForInstancesGone(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	ctx, cancel := context.WithTimeout(ctx, instanceDeletionTimeout)
	defer cancel()
	return wait.PollUntilContextCancel(ctx, 1*time.Second, true, func(ctx context.Context) (bool, error) {
		objs, err := listInstances(ctx, dynClient, gvr)
		if err != nil {
			return false, err
		}
		return len(objs) == 0, nil
	})
}

// waitForCRDDeletion blocks until the named CRD is gone or the timeout elapses.
func waitForCRDDeletion(ctx context.Context, crdClient apiextensionsclientset.Interface, name string) error {
	ctx, cancel := context.WithTimeout(ctx, crdDeletionTimeout)
	defer cancel()
	err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return fmt.Errorf("waiting for CRD %s to be deleted: %w", name, err)
	}
	return nil
}
