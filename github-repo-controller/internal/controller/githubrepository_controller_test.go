/*
Copyright 2026.

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

package controller

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	repov1 "github.com/example/github-repo-controller/api/v1"
	ghfake "github.com/example/github-repo-controller/internal/github/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = repov1.AddToScheme(s)
	return s
}

func newRepo(name, ns string) *repov1.GitHubRepository {
	return &repov1.GitHubRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: repov1.GitHubRepositorySpec{
			Owner:      "myorg",
			Name:       "test",
			Visibility: "public",
		},
	}
}

func TestReconcileSetsReady(t *testing.T) {
	ctx := context.Background()
	repo := newRepo("test-repo", "default")
	scheme := newScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(repo).Build()

	r := &GitHubRepositoryReconciler{
		Client:         client,
		Scheme:         scheme,
		PlatformClient: ghfake.NewClient(),
	}

	_, err := r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Name: repo.Name, Namespace: repo.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	updated := &repov1.GitHubRepository{}
	if err := client.Get(ctx, types.NamespacedName{Name: repo.Name, Namespace: repo.Namespace}, updated); err != nil {
		t.Fatalf("failed to get updated repo: %v", err)
	}

	if updated.Status.Phase != repov1.PhaseReady {
		t.Errorf("expected phase %q, got %q", repov1.PhaseReady, updated.Status.Phase)
	}
	if updated.Status.URL != "https://github.com/myorg/test" {
		t.Errorf("expected URL %q, got %q", "https://github.com/myorg/test", updated.Status.URL)
	}
	if !meta.IsStatusConditionTrue(updated.Status.Conditions, "Ready") {
		t.Errorf("expected Ready condition to be True, got %v", updated.Status.Conditions)
	}
}

func TestReconcileFailsWithoutPlatformClient(t *testing.T) {
	ctx := context.Background()
	repo := newRepo("test-repo", "default")
	scheme := newScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(repo).Build()

	r := &GitHubRepositoryReconciler{
		Client: client,
		Scheme: scheme,
	}

	_, err := r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Name: repo.Name, Namespace: repo.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	updated := &repov1.GitHubRepository{}
	if err := client.Get(ctx, types.NamespacedName{Name: repo.Name, Namespace: repo.Namespace}, updated); err != nil {
		t.Fatalf("failed to get updated repo: %v", err)
	}

	if updated.Status.Phase != repov1.PhaseFailed {
		t.Errorf("expected phase %q, got %q", repov1.PhaseFailed, updated.Status.Phase)
	}
}
