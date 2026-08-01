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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepositoryPhase describes the lifecycle phase of a GitHubRepository.
// +kubebuilder:validation:Enum=Pending;Reconciling;Ready;Failed
// +kubebuilder:validation:Type=string
type RepositoryPhase string

const (
	PhasePending     RepositoryPhase = "Pending"
	PhaseReconciling RepositoryPhase = "Reconciling"
	PhaseReady       RepositoryPhase = "Ready"
	PhaseFailed      RepositoryPhase = "Failed"
)

// GitHubRepositorySpec defines the desired state of GitHubRepository.
type GitHubRepositorySpec struct {
	// Owner is the GitHub user or organization that owns the repository.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Owner string `json:"owner"`

	// Name is the repository name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is an optional repository description.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Visibility is the repository visibility. Must be "public" or "private".
	// +kubebuilder:validation:Enum=public;private
	// +kubebuilder:default=public
	// +kubebuilder:validation:Optional
	Visibility string `json:"visibility,omitempty"`

	// InitializeReadme creates a README.md on repository creation.
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	InitializeReadme bool `json:"initializeReadme,omitempty"`

	// LicenseTemplate is an optional license template (e.g., "mit", "apache-2.0").
	// +kubebuilder:validation:Optional
	LicenseTemplate string `json:"licenseTemplate,omitempty"`
}

// GitHubRepositoryStatus defines the observed state of GitHubRepository.
type GitHubRepositoryStatus struct {
	// URL is the canonical web URL of the repository.
	// +kubebuilder:validation:Optional
	URL string `json:"url,omitempty"`

	// Phase is the current lifecycle phase of the repository.
	// +kubebuilder:validation:Optional
	Phase RepositoryPhase `json:"phase,omitempty"`

	// ObservedGeneration is the last observed metadata.generation.
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +kubebuilder:validation:Optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ghr
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=".spec.owner"
// +kubebuilder:printcolumn:name="Repo",type=string,JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Visibility",type=string,JSONPath=".spec.visibility"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// GitHubRepository is the Schema for the githubrepositories API.
type GitHubRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHubRepositorySpec   `json:"spec,omitempty"`
	Status GitHubRepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitHubRepositoryList contains a list of GitHubRepository.
type GitHubRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHubRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitHubRepository{}, &GitHubRepositoryList{})
}
