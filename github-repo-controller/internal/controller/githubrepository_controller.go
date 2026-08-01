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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	repov1 "github.com/example/github-repo-controller/api/v1"
	"github.com/example/github-repo-controller/internal/github"
)

const (
	githubRepoFinalizer = "repo.example.com/finalizer"
	requeueAfter        = 30 * time.Second
)

// GitHubRepositoryReconciler reconciles a GitHubRepository object.
type GitHubRepositoryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// PlatformClient talks to GitHub. A fake implementation can be injected
	// in unit tests.
	PlatformClient github.Client
}

//+kubebuilder:rbac:groups=repo.example.com,resources=githubrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo.example.com,resources=githubrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo.example.com,resources=githubrepositories/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile moves the current cluster state closer to the desired state.
func (r *GitHubRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("githubrepository", req.NamespacedName)

	var repo repov1.GitHubRepository
	if err := r.Get(ctx, req.NamespacedName, &repo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Defer a status update so every exit path persists the latest state.
	defer func() {
		if err := r.Status().Update(ctx, &repo); err != nil {
			logger.Error(err, "failed to update GitHubRepository status")
		}
	}()

	// Handle deletion and finalizer cleanup.
	if !repo.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &repo)
	}

	// Ensure the finalizer is present so we can clean up the GitHub repo on deletion.
	if !controllerutil.ContainsFinalizer(&repo, githubRepoFinalizer) {
		controllerutil.AddFinalizer(&repo, githubRepoFinalizer)
		if err := r.Update(ctx, &repo); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	return r.reconcileNormal(ctx, &repo)
}

func (r *GitHubRepositoryReconciler) reconcileNormal(ctx context.Context, repo *repov1.GitHubRepository) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if r.PlatformClient == nil {
		setCondition(repo, "Ready", metav1.ConditionFalse, "NoPlatformClient", "Platform client is not configured")
		repo.Status.Phase = repov1.PhaseFailed
		return ctrl.Result{}, nil
	}

	repo.Status.Phase = repov1.PhaseReconciling
	setCondition(repo, "Ready", metav1.ConditionFalse, "Reconciling", "Reconciliation in progress")

	cfg := &github.RepositoryConfig{
		Description:      repo.Spec.Description,
		Visibility:       repo.Spec.Visibility,
		InitializeReadme: repo.Spec.InitializeReadme,
		LicenseTemplate:  repo.Spec.LicenseTemplate,
	}

	info, err := r.PlatformClient.CreateOrUpdateRepository(ctx, repo.Spec.Owner, repo.Spec.Name, cfg)
	if err != nil {
		logger.Error(err, "failed to reconcile GitHub repository")
		if r.Recorder != nil {
			r.Recorder.Eventf(repo, "Warning", "RepositoryReconcileFailed", "GitHub API error: %v", err)
		}
		repo.Status.Phase = repov1.PhaseFailed
		setCondition(repo, "Ready", metav1.ConditionFalse, "RepositoryReconcileFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	repo.Status.URL = info.URL
	repo.Status.Phase = repov1.PhaseReady
	repo.Status.ObservedGeneration = repo.Generation
	setCondition(repo, "Ready", metav1.ConditionTrue, "Success", "Repository reconciled")
	if r.Recorder != nil {
		r.Recorder.Eventf(repo, "Normal", "RepositoryReconciled", "Repository available at %s", info.URL)
	}

	return ctrl.Result{}, nil
}

func (r *GitHubRepositoryReconciler) reconcileDelete(ctx context.Context, repo *repov1.GitHubRepository) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(repo, githubRepoFinalizer) {
		return ctrl.Result{}, nil
	}

	if r.PlatformClient == nil {
		logger.Info("no platform client configured, skipping external repository deletion")
	} else if err := r.PlatformClient.DeleteRepository(ctx, repo.Spec.Owner, repo.Spec.Name); err != nil {
		logger.Error(err, "failed to delete GitHub repository")
		if r.Recorder != nil {
			r.Recorder.Eventf(repo, "Warning", "RepositoryDeleteFailed", "GitHub API error: %v", err)
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	controllerutil.RemoveFinalizer(repo, githubRepoFinalizer)
	if err := r.Update(ctx, repo); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitHubRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("githubrepository-controller")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&repov1.GitHubRepository{}).
		Complete(r)
}

func setCondition(repo *repov1.GitHubRepository, typ string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&repo.Status.Conditions, metav1.Condition{
		Type:               typ,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: repo.Generation,
	})
}
