# GitHub Repository Controller

A Kubernetes controller that manages GitHub repositories via a custom resource.

## Overview

The controller watches `GitHubRepository` resources in the cluster and ensures
that a matching repository exists on GitHub with the desired configuration.

## Custom Resource

```yaml
apiVersion: repo.example.com/v1
kind: GitHubRepository
metadata:
  name: sample
  namespace: default
spec:
  owner: myorg
  name: sample
  description: A sample repository managed by Kubernetes
  visibility: public
  initializeReadme: true
  licenseTemplate: mit
```

### Spec Fields

| Field | Description |
|-------|-------------|
| `owner` | GitHub user or organization that owns the repository (required) |
| `name` | Repository name (required) |
| `description` | Optional repository description |
| `visibility` | `public` or `private` (default: `public`) |
| `initializeReadme` | Whether to create a README on repository creation |
| `licenseTemplate` | Optional license template, e.g. `mit`, `apache-2.0` |

### Status Fields

| Field | Description |
|-------|-------------|
| `url` | Canonical web URL of the repository |
| `phase` | `Pending`, `Reconciling`, `Ready`, or `Failed` |
| `observedGeneration` | Last observed `metadata.generation` |
| `conditions` | Standard Kubernetes conditions |

## Project Layout

```
.
├── api/v1/                          # CRD Go types
├── cmd/main.go                      # Manager entrypoint
├── config/
│   ├── crd/bases/                   # Generated CRD YAML
│   ├── samples/                     # Sample CRs
│   ├── manager/                     # Deployment manifests
│   └── rbac/                        # Generated RBAC
├── internal/
│   ├── controller/                  # Reconcile logic
│   ├── github/client.go             # GitHub client interface
│   └── github/fake/fake.go          # Fake client for tests
├── go.mod
├── Makefile
└── Dockerfile
```

## Development

```bash
# Build the manager binary
make build

# Generate DeepCopy methods
make generate

# Generate CRDs and RBAC
make manifests

# Run unit tests (fake client, no cluster needed)
go test ./...

# Run the controller locally (requires a kubeconfig and GitHub token integration)
make run
```

## Next Steps

1. Implement a real `github.Client` that calls the GitHub REST API.
2. Pass the client into the reconciler in `cmd/main.go`.
3. Add a `GitHubToken` secret reference to the CRD or load credentials from env.
4. Install the CRD into a cluster:
   ```bash
   make install
   ```
5. Deploy the controller:
   ```bash
   make deploy IMG=myregistry/github-repo-controller:v0.1.0
   ```

## Safety

- `make install/deploy` apply resources to the current kubeconfig context.
- Always review generated manifests before applying to shared clusters.
- Never commit GitHub tokens; use Kubernetes Secrets or an external secret manager.
