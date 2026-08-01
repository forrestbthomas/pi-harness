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

package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/github-repo-controller/internal/github"
)

// Client is a thread-safe fake implementation of github.Client for tests.
type Client struct {
	mu     sync.Mutex
	Repos  map[string]*github.RepositoryInfo
	Err    error
	Deleted map[string]bool
}

// NewClient returns a fake client with an empty repository map.
func NewClient() *Client {
	return &Client{
		Repos:   make(map[string]*github.RepositoryInfo),
		Deleted: make(map[string]bool),
	}
}

func key(owner, name string) string {
	return fmt.Sprintf("%s/%s", owner, name)
}

// CreateOrUpdateRepository records the desired repository and returns a stub info.
func (c *Client) CreateOrUpdateRepository(ctx context.Context, owner, name string, cfg *github.RepositoryConfig) (*github.RepositoryInfo, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	k := key(owner, name)
	info := &github.RepositoryInfo{
		URL:        fmt.Sprintf("https://github.com/%s", k),
		Visibility: cfg.Visibility,
	}
	c.Repos[k] = info
	return info, nil
}

// DeleteRepository records the deletion.
func (c *Client) DeleteRepository(ctx context.Context, owner, name string) error {
	if c.Err != nil {
		return c.Err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Deleted[key(owner, name)] = true
	return nil
}
