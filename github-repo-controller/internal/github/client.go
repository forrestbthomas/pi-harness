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

// Package github abstracts interactions with the GitHub API so the
// controller can be tested without making real network calls.
package github

import "context"

// RepositoryConfig carries the desired repository settings.
type RepositoryConfig struct {
	Description      string
	Visibility       string
	InitializeReadme bool
	LicenseTemplate  string
}

// RepositoryInfo carries observed repository settings returned by the platform.
type RepositoryInfo struct {
	URL        string
	Visibility string
}

// Client is the interface the controller uses to manage repositories.
// Production implementations call the GitHub REST/GraphQL API; tests can
// inject a fake implementation.
type Client interface {
	// CreateOrUpdateRepository ensures a repository with the given owner/name
	// exists and matches the supplied configuration. It returns the observed
	// repository state.
	CreateOrUpdateRepository(ctx context.Context, owner, name string, cfg *RepositoryConfig) (*RepositoryInfo, error)

	// DeleteRepository removes a repository. It should be idempotent and not
	// return an error if the repository does not exist.
	DeleteRepository(ctx context.Context, owner, name string) error
}
