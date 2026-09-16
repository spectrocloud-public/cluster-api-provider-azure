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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testCRDName  = "fleetsmembers.containerservice.azure.com"
	testGroup    = "containerservice.azure.com"
	testPlural   = "fleetsmembers"
	testStale    = "v1api20230315previewstorage"
	testServed   = "v1api20230315preview"
	testNewStore = "v1api20250301"
	// testNamespace is the ASO pod namespace the namespace fixture names.
	testNamespace = "capi-webhook-system"
)

func testCRD(storedVersions []string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: testCRDName},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: testGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: testPlural, Kind: "FleetsMember"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: testServed, Served: true},
				{Name: testStale, Served: false, Storage: true},
				{Name: testNewStore, Served: true},
			},
		},
		Status: apiextensionsv1.CustomResourceDefinitionStatus{StoredVersions: storedVersions},
	}
}

// testDynamicClient registers list kinds for every CRD version so the fake can
// list each <plural>.<version>.<group> without erroring; instances are created
// only under the served version.
func testDynamicClient(instances int) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{}
	for _, v := range []string{testServed, testStale, testNewStore} {
		gvr := schema.GroupVersionResource{Group: testGroup, Version: v, Resource: testPlural}
		gvrToListKind[gvr] = "FleetsMemberList"
	}
	objs := make([]runtime.Object, 0, instances)
	for i := range instances {
		objs = append(objs, &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": testGroup + "/" + testServed,
			"kind":       "FleetsMember",
			"metadata": map[string]interface{}{
				"name":      "member-" + string(rune('a'+i)),
				"namespace": "default",
			},
		}})
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, objs...)
}

func crdExists(g *WithT, crdClient *apiextensionsfake.Clientset, name string) bool {
	_, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false
	}
	g.Expect(err).NotTo(HaveOccurred())
	return true
}

func TestMigrateASOCRDStoredVersion(t *testing.T) {
	migration := storedVersionMigration{crd: testCRDName, stale: testStale}
	ctx := context.Background()
	logger := log.Log

	t.Run("skips when CRD is absent", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset()
		err := migrateStoredVersion(ctx, logger, crdClient, testDynamicClient(0), migration)
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("skips when stale version is not stored", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(testCRD([]string{testNewStore}))
		err := migrateStoredVersion(ctx, logger, crdClient, testDynamicClient(0), migration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(crdExists(g, crdClient, testCRDName)).To(BeTrue(), "CRD must be left intact")
	})

	t.Run("deletes the CRD when stale and empty", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(testCRD([]string{testStale, testNewStore}))
		err := migrateStoredVersion(ctx, logger, crdClient, testDynamicClient(0), migration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(crdExists(g, crdClient, testCRDName)).To(BeFalse(), "CRD must be deleted")
	})

	t.Run("fails when stale but instances exist", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(testCRD([]string{testStale, testNewStore}))
		err := migrateStoredVersion(ctx, logger, crdClient, testDynamicClient(2), migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("2 instance(s)"))
		g.Expect(err.Error()).To(ContainSubstring("asoctl clean crds"))
		g.Expect(crdExists(g, crdClient, testCRDName)).To(BeTrue(), "CRD must not be deleted while it holds data")
	})

	t.Run("fails closed when listing a served version errors", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(testCRD([]string{testStale, testNewStore}))
		dynClient := testDynamicClient(0)
		dynClient.PrependReactor("list", testPlural, func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(errors.New("conversion webhook down"))
		})
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(crdExists(g, crdClient, testCRDName)).To(BeTrue(), "CRD must not be deleted when emptiness can't be confirmed")
	})
}

// The managedclusters fixtures below mirror the CRD shape found on clusters
// upgraded from the CAPZ v1.18 ASO bundle: exactly one stored version, which
// is both served and the storage version. In that state a list at the stale
// version decodes the etcd-native encoding and must never involve the ASO
// conversion webhook (which is unreachable inside the init container); the
// tests assert that only that version is ever listed.
const (
	mcCRDName = "managedclusters.containerservice.azure.com"
	mcGroup   = "containerservice.azure.com"
	mcPlural  = "managedclusters"
	mcKind    = "ManagedCluster"
	mcStale   = "v1api20231001storage"
	mcServed  = "v1api20231001"
)

func managedClustersCRD(storedVersions []string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: mcCRDName},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: mcGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: mcPlural, Kind: mcKind},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: mcServed, Served: true},
				{Name: mcStale, Served: true, Storage: true},
			},
		},
		Status: apiextensionsv1.CustomResourceDefinitionStatus{StoredVersions: storedVersions},
	}
}

// mcInstance builds a ManagedCluster instance in the stored encoding, with an
// optional ownerReference and finalizer set.
func mcInstance(name string, ownerRef map[string]interface{}, finalizers []string) *unstructured.Unstructured {
	meta := map[string]interface{}{"name": name, "namespace": "default"}
	if len(finalizers) > 0 {
		refs := make([]interface{}, 0, len(finalizers))
		for _, f := range finalizers {
			refs = append(refs, f)
		}
		meta["finalizers"] = refs
	}
	if ownerRef != nil {
		meta["ownerReferences"] = []interface{}{ownerRef}
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mcGroup + "/" + mcStale,
		"kind":       mcKind,
		"metadata":   meta,
	}}
}

func capzOwnerRef(name string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": capzOwnerGroup + "/v1beta1",
		"kind":       "AzureManagedControlPlane",
		"name":       name,
		"uid":        "uid-" + name,
	}
}

func asoOwnerRef(name string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": mcGroup + "/" + mcServed,
		"kind":       mcKind,
		"name":       name,
		"uid":        "uid-" + name,
	}
}

// withUID stamps uid onto the object's metadata. The fake dynamic client
// round-trips it verbatim, so the ownership checks in the code under test see
// a realistic UID.
func withUID(obj *unstructured.Unstructured, uid string) *unstructured.Unstructured {
	obj.SetUID(types.UID(uid))
	return obj
}

// withOperatorSpec declares Secret and ConfigMap destinations in the object's
// spec.operatorSpec, the shape CAPZ's managedclusters controllers set: each
// destination key maps to the name of the core object ASO should write.
func withOperatorSpec(obj *unstructured.Unstructured, secrets, configMaps map[string]string) *unstructured.Unstructured {
	spec := map[string]interface{}{}
	if len(secrets) > 0 {
		s := map[string]interface{}{}
		for k, name := range secrets {
			s[k] = map[string]interface{}{"name": name, "key": "value"}
		}
		spec["secrets"] = s
	}
	if len(configMaps) > 0 {
		c := map[string]interface{}{}
		for k, name := range configMaps {
			c[k] = map[string]interface{}{"name": name, "key": "value"}
		}
		spec["configMaps"] = c
	}
	if len(spec) > 0 {
		obj.Object["spec"] = map[string]interface{}{"operatorSpec": spec}
	}
	return obj
}

// asoOutput builds the Secret or ConfigMap ASO writes as a dependent of a CR:
// kind is "Secret" or "ConfigMap", and the object carries an ownerReference
// naming the ManagedCluster crName of the given UID in the stale storage
// apiVersion, the reference the garbage collector can no longer map once the
// CRD is replaced.
func asoOutput(kind, name, crName, ownerUID string) *unstructured.Unstructured {
	resource := map[string]string{"Secret": "secrets", "ConfigMap": "configmaps"}[kind]
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": "default",
			"ownerReferences": []interface{}{map[string]interface{}{
				"apiVersion": mcGroup + "/" + mcStale,
				"kind":       mcKind,
				"name":       crName,
				"uid":        ownerUID,
			}},
		},
		"data": map[string]interface{}{resource: "a3ViZWNvbmZpZw=="},
	}}
}

// outputExists reports whether the named core object still exists.
func outputExists(g *WithT, dynClient *dynamicfake.FakeDynamicClient, kind, name string) bool {
	resource := map[string]string{"Secret": "secrets", "ConfigMap": "configmaps"}[kind]
	_, err := dynClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: resource}).
		Namespace("default").Get(context.Background(), name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false
	}
	g.Expect(err).NotTo(HaveOccurred())
	return true
}

// leaseGVR is the resource the idleness check reads for ASO's leader-election
// lease.
var leaseGVR = schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}

// mcDynamicClient builds the fake client for the managedclusters fixtures: the
// CRD versions, the leader-election leases resource, and the core Secrets and
// ConfigMaps resources (the operatorSpec outputs ASO writes), with no lease
// present (the state of a namespace whose ASO manager has never run), seeded
// with any given instances.
func mcDynamicClient(objs ...*unstructured.Unstructured) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		leaseGVR: "LeaseList",
		{Group: "", Version: "v1", Resource: "secrets"}:    "SecretList",
		{Group: "", Version: "v1", Resource: "configmaps"}: "ConfigMapList",
	}
	for _, v := range []string{mcServed, mcStale} {
		gvrToListKind[schema.GroupVersionResource{Group: mcGroup, Version: v, Resource: mcPlural}] = mcKind + "List"
	}
	runtimeObjs := make([]runtime.Object, 0, len(objs))
	for _, o := range objs {
		runtimeObjs = append(runtimeObjs, o)
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, runtimeObjs...)
}

// mcDynamicClientWithLease is mcDynamicClient with a leader-election lease
// seeded into it.
func mcDynamicClientWithLease(lease *unstructured.Unstructured, objs ...*unstructured.Unstructured) *dynamicfake.FakeDynamicClient {
	client := mcDynamicClient(objs...)
	if lease != nil {
		if err := client.Tracker().Add(lease); err != nil {
			panic(err)
		}
	}
	return client
}

// leaseFixture builds a leader-election lease named exactly as ASO's manager
// names it, with the given spec.
func leaseFixture(spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": leaseGVR.GroupVersion().String(),
		"kind":       "Lease",
		"metadata": map[string]interface{}{
			"name":      asoLeaderElectionLease,
			"namespace": testNamespace,
		},
		"spec": spec,
	}}
}

// runningLease builds a lease that looks actively renewed, as an ASO manager
// running right now would leave it.
func runningLease(holder string) *unstructured.Unstructured {
	return leaseFixture(map[string]interface{}{
		"holderIdentity":       holder,
		"leaseDurationSeconds": int64(15),
		"renewTime":            time.Now().UTC().Format(time.RFC3339),
	})
}

// useNamespaceFixture points the pod-namespace file the idleness check reads
// at a fixture holding ns. Register it on a parent test so it covers all
// subtests.
func useNamespaceFixture(t *testing.T, ns string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "namespace")
	if err := os.WriteFile(path, []byte(ns+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := serviceAccountNamespaceFile
	serviceAccountNamespaceFile = path
	t.Cleanup(func() { serviceAccountNamespaceFile = orig })
}

// listedVersions returns the API version every list action targeted.
func listedVersions(actions []clienttesting.Action) []string {
	var versions []string
	for _, a := range actions {
		if a.GetVerb() == "list" {
			versions = append(versions, a.GetResource().Version)
		}
	}
	return versions
}

func verbsUsed(actions []clienttesting.Action) map[string]bool {
	verbs := map[string]bool{}
	for _, a := range actions {
		verbs[a.GetVerb()] = true
	}
	return verbs
}

func TestMigrateASOCRDStoredVersionSoleStorageVersion(t *testing.T) {
	migration := storedVersionMigration{crd: mcCRDName, stale: mcStale}
	ctx := context.Background()
	logger := log.Log
	useNamespaceFixture(t, testNamespace)

	t.Run("deletes the CRD when the storage version is empty", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient()
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeFalse(), "CRD must be deleted")
		g.Expect(listedVersions(dynClient.Actions())).To(ConsistOf(mcStale),
			"only the storage version may be listed; listing the served non-storage version would go through the conversion webhook")
	})

	t.Run("cleans up CAPZ-owned instances and deletes the CRD", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}),
			mcInstance("pool-b", capzOwnerRef("ammp-b"), []string{asoFinalizer, "custom.example.com/finalizer"}),
		)
		// No leader-election lease exists, which is the no-running-manager
		// proof: the cleanup must proceed without touching any conversion
		// webhook.
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeFalse(), "CRD must be deleted after draining instances")

		list, err := dynClient.Resource(schema.GroupVersionResource{Group: mcGroup, Version: mcStale, Resource: mcPlural}).List(ctx, metav1.ListOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(list.Items).To(BeEmpty(), "instances must be deleted")

		versions := listedVersions(dynClient.Actions())
		g.Expect(versions).NotTo(BeEmpty())
		g.Expect(versions).To(HaveEach(Equal(mcStale)),
			"only the storage encoding may ever be listed; nothing here may depend on the conversion webhook")

		deleteIdx := map[string]int{}
		for i, a := range dynClient.Actions() {
			if a.GetVerb() != "delete" {
				continue
			}
			da, ok := a.(clienttesting.DeleteAction)
			g.Expect(ok).To(BeTrue())
			deleteIdx[da.GetName()] = i
		}
		g.Expect(deleteIdx).To(HaveLen(2), "every CAPZ-owned instance must be deleted")

		// The ASO finalizer strip must precede the delete of the same object
		// and preserve unrelated finalizers.
		sawPoolBPatch := false
		for i, a := range dynClient.Actions() {
			if a.GetVerb() != "patch" {
				continue
			}
			pa, ok := a.(clienttesting.PatchAction)
			g.Expect(ok).To(BeTrue())
			if pa.GetName() != "pool-b" {
				continue
			}
			sawPoolBPatch = true
			var body map[string]interface{}
			g.Expect(json.Unmarshal(pa.GetPatch(), &body)).To(Succeed())
			finalizers := body["metadata"].(map[string]interface{})["finalizers"].([]interface{})
			g.Expect(finalizers).To(ConsistOf("custom.example.com/finalizer"),
				"only the ASO finalizer may be stripped")
			g.Expect(deleteIdx["pool-b"]).To(BeNumerically(">", i),
				"the finalizer must be removed before the object is deleted")
		}
		g.Expect(sawPoolBPatch).To(BeTrue(), "the ASO finalizer must be stripped before deletion")
	})

	t.Run("deletes the CR's operatorSpec outputs before deleting the CR", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			withOperatorSpec(withUID(mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}), "gen-2"),
				map[string]string{"adminCredentials": "cluster-a-aso-kubeconfig", "userCredentials": "cluster-a-aso-user-kubeconfig"},
				map[string]string{"oidcIssuerProfile": "cluster-a-oidc"},
			),
			// ASO wrote this Secret for the current generation of cluster-a.
			// Its ownerReference is healthy today, but the CR is about to be
			// deleted and recreated under a new UID, so the migration must
			// take the dependent with it.
			asoOutput("Secret", "cluster-a-aso-kubeconfig", "cluster-a", "gen-2"),
			asoOutput("ConfigMap", "cluster-a-oidc", "cluster-a", "gen-2"),
			// Not owned by the CR: ASO would refuse to write over it, and so
			// must the migration.
			&unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "v1", "kind": "Secret",
				"metadata": map[string]interface{}{"name": "user-owned", "namespace": "default"},
				"data":     map[string]interface{}{"secrets": "dXNlcg=="},
			}},
		)
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeFalse(), "CRD must be deleted after draining instances")

		g.Expect(outputExists(g, dynClient, "Secret", "cluster-a-aso-kubeconfig")).To(BeFalse(),
			"the kubeconfig Secret would be orphaned by the CR deletion and must be deleted with it")
		g.Expect(outputExists(g, dynClient, "ConfigMap", "cluster-a-oidc")).To(BeFalse(),
			"the OIDC ConfigMap would be orphaned by the CR deletion and must be deleted with it")
		g.Expect(outputExists(g, dynClient, "Secret", "user-owned")).To(BeTrue(),
			"an output without an ownerReference to the CR may be user-owned and must be left alone")

		// The declared userCredentials destination has no object; the migration
		// must not fail on it.
		deletes := map[string]int{}
		var crPatchIdx, crDeleteIdx = -1, -1
		for i, a := range dynClient.Actions() {
			if a.GetVerb() == "delete" {
				da, ok := a.(clienttesting.DeleteAction)
				g.Expect(ok).To(BeTrue())
				deletes[da.GetResource().Resource+"/"+da.GetName()] = i
			}
			if a.GetVerb() == "patch" && a.GetResource().Resource == mcPlural {
				crPatchIdx = i
			}
			if a.GetVerb() == "delete" && a.GetResource().Resource == mcPlural {
				crDeleteIdx = i
			}
		}
		g.Expect(deletes).To(HaveKey("secrets/cluster-a-aso-kubeconfig"))
		g.Expect(deletes).To(HaveKey("configmaps/cluster-a-oidc"))
		g.Expect(deletes).NotTo(HaveKey("secrets/cluster-a-aso-user-kubeconfig"))
		g.Expect(deletes).NotTo(HaveKey("secrets/user-owned"))
		g.Expect(deletes["secrets/cluster-a-aso-kubeconfig"]).To(BeNumerically("<", crPatchIdx),
			"outputs must be deleted before the CR's finalizer is stripped")
		g.Expect(deletes["secrets/cluster-a-aso-kubeconfig"]).To(BeNumerically("<", crDeleteIdx),
			"outputs must be deleted before the CR itself is deleted")

		for _, a := range dynClient.Actions() {
			if a.GetVerb() == "list" && a.GetResource().Resource == mcPlural {
				g.Expect(a.GetResource().Version).To(Equal(mcStale),
					"only the storage encoding of the CR may be listed; nothing here may depend on the conversion webhook")
			}
		}
	})

	t.Run("fails closed when an operatorSpec output cannot be deleted", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			withOperatorSpec(withUID(mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}), "gen-2"),
				map[string]string{"adminCredentials": "cluster-a-aso-kubeconfig"}, nil,
			),
			asoOutput("Secret", "cluster-a-aso-kubeconfig", "cluster-a", "gen-2"),
		)
		dynClient.PrependReactor("delete", "secrets", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(errors.New("etcd timeout"))
		})
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("operatorSpec output of ManagedCluster cluster-a/default being deleted"))
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeTrue(),
			"CRD must not be deleted when its instances' outputs cannot be cleaned up")
		verbs := verbsUsed(dynClient.Actions())
		g.Expect(verbs).NotTo(HaveKey("patch"), "no finalizer may be stripped when output cleanup fails")
		list, listErr := dynClient.Resource(schema.GroupVersionResource{Group: mcGroup, Version: mcStale, Resource: mcPlural}).List(ctx, metav1.ListOptions{})
		g.Expect(listErr).NotTo(HaveOccurred())
		g.Expect(list.Items).To(HaveLen(1), "the CR must be left in place when output cleanup fails")
	})

	t.Run("fails closed when any instance is not CAPZ-owned", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}),
			mcInstance("hand-made", nil, []string{asoFinalizer}),
		)
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not owned by a cluster-api-provider-azure resource"))
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeTrue(),
			"CRD must not be deleted while foreign instances remain")
		verbs := verbsUsed(dynClient.Actions())
		g.Expect(verbs).NotTo(HaveKey("delete"), "no instance may be deleted when foreign instances remain")
		g.Expect(verbs).NotTo(HaveKey("patch"), "no finalizer may be stripped when foreign instances remain")
	})

	t.Run("fails closed when all instances are foreign", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			mcInstance("external-a", asoOwnerRef("someone-else"), []string{asoFinalizer}),
		)
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not owned by a cluster-api-provider-azure resource"))
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeTrue(),
			"CRD must not be deleted while foreign instances remain")
		list, listErr := dynClient.Resource(schema.GroupVersionResource{Group: mcGroup, Version: mcStale, Resource: mcPlural}).List(ctx, metav1.ListOptions{})
		g.Expect(listErr).NotTo(HaveOccurred())
		g.Expect(list.Items).To(HaveLen(1), "the foreign instance must be left untouched")
		g.Expect(list.Items[0].GetFinalizers()).To(ContainElement(asoFinalizer),
			"no finalizer may be stripped when all instances are foreign")
	})

	t.Run("refuses to clean up while the ASO leader-election lease is fresh", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClientWithLease(
			runningLease("capz-azureserviceoperator-controller-manager-745ccdc6bc-68tcj"),
			mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}),
		)
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("controllers are still running"))
		g.Expect(err.Error()).To(ContainSubstring("745ccdc6bc-68tcj"),
			"the message must name the holder so a human can find the running pod")
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeTrue(),
			"CRD must not be deleted while an ASO manager holds the lease")
		verbs := verbsUsed(dynClient.Actions())
		g.Expect(verbs).NotTo(HaveKey("delete"), "no instance may be deleted while an ASO manager holds the lease")
		g.Expect(verbs).NotTo(HaveKey("patch"), "no finalizer may be stripped while an ASO manager holds the lease")
	})

	t.Run("fails closed when the leader-election lease cannot be read", func(t *testing.T) {
		g := NewWithT(t)
		crdClient := apiextensionsfake.NewSimpleClientset(managedClustersCRD([]string{mcStale}))
		dynClient := mcDynamicClient(
			mcInstance("cluster-a", capzOwnerRef("amcp-a"), []string{asoFinalizer}),
		)
		dynClient.PrependReactor("get", "leases", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				schema.GroupResource{Group: leaseGVR.Group, Resource: leaseGVR.Resource}, "", errors.New("denied"))
		})
		err := migrateStoredVersion(ctx, logger, crdClient, dynClient, migration)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("leader-election lease"))
		g.Expect(crdExists(g, crdClient, mcCRDName)).To(BeTrue(),
			"CRD must not be deleted when safety cannot be established")
		verbs := verbsUsed(dynClient.Actions())
		g.Expect(verbs).NotTo(HaveKey("delete"), "no instance may be deleted when safety cannot be established")
		g.Expect(verbs).NotTo(HaveKey("patch"), "no finalizer may be stripped when safety cannot be established")
	})
}

// TestConfirmASOControllersNotRunning covers the lease states the idleness
// check decides on. The lease fixtures use the real lease name and realistic
// holder identities; the staleness threshold is the 15s lease duration plus
// the 60s clock-skew margin, so the table's "inside" and "outside" cases sit
// well clear of the boundary.
func TestConfirmASOControllersNotRunning(t *testing.T) {
	ctx := context.Background()
	mcGVR := schema.GroupVersionResource{Group: mcGroup, Version: mcStale, Resource: mcPlural}
	useNamespaceFixture(t, testNamespace)

	tests := []struct {
		name    string
		lease   *unstructured.Unstructured
		wantErr string
	}{
		{
			name:  "no lease means no manager has ever led",
			lease: nil,
		},
		{
			name:  "a released lease, holder cleared on graceful shutdown, is safe",
			lease: leaseFixture(map[string]interface{}{"renewTime": time.Now().UTC().Format(time.RFC3339)}),
		},
		{
			name: "a lease not renewed for its full duration is stale",
			lease: leaseFixture(map[string]interface{}{
				"holderIdentity":       "capz-azureserviceoperator-controller-manager-745ccdc6bc-68tcj",
				"leaseDurationSeconds": int64(15),
				"renewTime":            time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
			}),
		},
		{
			name: "a stale lease without a duration falls back to the 15s default",
			lease: leaseFixture(map[string]interface{}{
				"holderIdentity": "capz-azureserviceoperator-controller-manager-745ccdc6bc-68tcj",
				"renewTime":      time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339),
			}),
		},
		{
			name: "a renewal inside the threshold means a manager is running",
			lease: leaseFixture(map[string]interface{}{
				"holderIdentity":       "capz-azureserviceoperator-controller-manager-745ccdc6bc-68tcj",
				"leaseDurationSeconds": int64(15),
				"renewTime":            time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339),
			}),
			wantErr: "controllers are still running",
		},
		{
			name:    "a holder without a renewTime fails closed",
			lease:   leaseFixture(map[string]interface{}{"holderIdentity": "some-manager"}),
			wantErr: "carries no renewTime",
		},
		{
			name: "an unparseable renewTime fails closed",
			lease: leaseFixture(map[string]interface{}{
				"holderIdentity": "capz-azureserviceoperator-controller-manager-745ccdc6bc-68tcj",
				"renewTime":      "not-a-time",
			}),
			wantErr: "parsing the renewTime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			dynClient := mcDynamicClientWithLease(tt.lease)
			err := confirmASOControllersNotRunning(ctx, dynClient, mcGVR)
			if tt.wantErr == "" {
				g.Expect(err).NotTo(HaveOccurred())
				return
			}
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
		})
	}

	t.Run("a lease read error fails closed", func(t *testing.T) {
		g := NewWithT(t)
		dynClient := mcDynamicClient()
		dynClient.PrependReactor("get", "leases", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				schema.GroupResource{Group: leaseGVR.Group, Resource: leaseGVR.Resource}, "", errors.New("denied"))
		})
		err := confirmASOControllersNotRunning(ctx, dynClient, mcGVR)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("leader-election lease"))
	})

	t.Run("a missing pod namespace fails closed", func(t *testing.T) {
		g := NewWithT(t)
		orig := serviceAccountNamespaceFile
		serviceAccountNamespaceFile = filepath.Join(t.TempDir(), "missing")
		t.Cleanup(func() { serviceAccountNamespaceFile = orig })
		err := confirmASOControllersNotRunning(ctx, mcDynamicClient(), mcGVR)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("determining the namespace of this ASO pod"))
	})
}

func TestIsCAPZOwned(t *testing.T) {
	tests := []struct {
		name      string
		ownerRefs []metav1.OwnerReference
		want      bool
	}{
		{name: "no owner references", want: false},
		{
			name:      "CAPZ owner reference",
			ownerRefs: []metav1.OwnerReference{{APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1", Kind: "AzureManagedControlPlane", Name: "c", UID: "u"}},
			want:      true,
		},
		{
			name:      "ASO owner reference only",
			ownerRefs: []metav1.OwnerReference{{APIVersion: "containerservice.azure.com/v1api20231001", Kind: "ManagedCluster", Name: "c", UID: "u"}},
			want:      false,
		},
		{
			name: "mixed owner references",
			ownerRefs: []metav1.OwnerReference{
				{APIVersion: "containerservice.azure.com/v1api20231001", Kind: "ManagedCluster", Name: "c", UID: "u"},
				{APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1", Kind: "AzureManagedMachinePool", Name: "p", UID: "v"},
			},
			want: true,
		},
		{
			name:      "core group owner reference",
			ownerRefs: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: "c", UID: "u"}},
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			obj := &unstructured.Unstructured{}
			obj.SetOwnerReferences(tt.ownerRefs)
			g.Expect(isCAPZOwned(obj)).To(Equal(tt.want))
		})
	}
}

func TestOperatorSpecDependencies(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want []operatorSpecDependency
	}{
		{
			name: "secrets before configmaps, each group in sorted key order",
			obj: withOperatorSpec(&unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": mcGroup + "/" + mcStale, "kind": mcKind,
				"metadata": map[string]interface{}{"name": "c", "namespace": "default"},
			}},
				map[string]string{"userCredentials": "c-user", "adminCredentials": "c-admin"},
				map[string]string{"oidcIssuerProfile": "c-oidc"},
			),
			want: []operatorSpecDependency{
				{name: "c-admin", kind: "Secret"},
				{name: "c-user", kind: "Secret"},
				{name: "c-oidc", kind: "ConfigMap"},
			},
		},
		{
			name: "no operatorSpec",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": mcGroup + "/" + mcStale, "kind": mcKind,
				"metadata": map[string]interface{}{"name": "c", "namespace": "default"},
			}},
			want: nil,
		},
		{
			name: "entries without a name are skipped",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": mcGroup + "/" + mcStale, "kind": mcKind,
				"metadata": map[string]interface{}{"name": "c", "namespace": "default"},
				"spec": map[string]interface{}{"operatorSpec": map[string]interface{}{
					"secrets": map[string]interface{}{
						"good":      map[string]interface{}{"name": "c-good", "key": "value"},
						"broken":    map[string]interface{}{"key": "value"},
						"notamap":   "nope",
						"emptyName": map[string]interface{}{"name": "", "key": "value"},
					},
				}},
			}},
			want: []operatorSpecDependency{{name: "c-good", kind: "Secret"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(operatorSpecDependencies(tt.obj)).To(Equal(tt.want))
		})
	}
}
