# Add Application

This guide covers deploying a standalone application to your platform. This is the kubarax equivalent of adding an ArgoCD `Application` in kubara.

For platform-level services deployed across clusters, see [Add HelmRelease](add-helmrelease.md) instead.

## Prerequisites

- The application's source repository has been [added](add-repository.md)
- A tenant project/namespace has been [created](add-project.md) if needed

## Option A: Deploy from a Git Repository (Kustomize/Plain YAML)

### Step 1: Create a Kustomization

Add a `Kustomization` that points to the application's manifests in a Git repository:

```yaml
# customer-service-catalog/helm/<cluster-name>/apps/my-app.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: my-app
  namespace: flux-system
spec:
  interval: 5m
  sourceRef:
    kind: GitRepository
    name: my-app-repo           # Must exist as a GitRepository source
  path: ./deploy/kubernetes      # Path within the repo
  targetNamespace: team-alpha    # Deploy into this namespace
  prune: true
  wait: true
  timeout: 5m
```

### Step 2: Add to Kustomization

```yaml
# customer-service-catalog/helm/<cluster-name>/kustomization.yaml
resources:
  # ... existing resources ...
  - apps/my-app.yaml
```

### Step 3: Commit and Push

```bash
git add .
git commit -m "feat: add my-app deployment"
git push
```

## Option B: Deploy from a Helm Chart

### Step 1: Create a HelmRelease

```yaml
# customer-service-catalog/helm/<cluster-name>/apps/my-helm-app.yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: my-helm-app
  namespace: flux-system
spec:
  interval: 10m
  chart:
    spec:
      chart: my-helm-app
      version: "1.2.x"
      sourceRef:
        kind: HelmRepository
        name: my-org-charts       # Must exist as a HelmRepository source
      interval: 1h
  targetNamespace: team-alpha
  install:
    createNamespace: false         # Namespace managed by tenant setup
    remediation:
      retries: 3
  upgrade:
    remediation:
      retries: 3
  values:
    replicaCount: 2
    image:
      repository: ghcr.io/org/my-helm-app
      tag: latest
    ingress:
      enabled: true
      className: traefik
      hosts:
        - host: my-app.platform.example.com
          paths:
            - path: /
              pathType: Prefix
```

## Option C: Deploy to a Worker Cluster

To deploy an application to a remote worker cluster from the control plane:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: my-app-on-worker
  namespace: flux-system
spec:
  interval: 5m
  sourceRef:
    kind: GitRepository
    name: my-app-repo
  path: ./deploy/kubernetes
  targetNamespace: team-alpha
  prune: true
  kubeConfig:
    secretRef:
      name: worker-0-kubeconfig   # Remote cluster credentials
```

## Option D: Deploy with Developer Self-Service

For developer self-service, use a `ResourceSet` with `ResourceSetInputProvider` to automatically deploy from Git branches or pull requests:

```yaml
apiVersion: fluxcd.controlplane.io/v1
kind: ResourceSetInputProvider
metadata:
  name: my-app-prs
  namespace: flux-system
spec:
  type: GitHubPullRequest
  url: https://github.com/org/my-app
  secretRef:
    name: github-token
  filter:
    labels: ["deploy/preview"]
---
apiVersion: fluxcd.controlplane.io/v1
kind: ResourceSet
metadata:
  name: my-app-previews
  namespace: flux-system
spec:
  inputsFrom:
    - apiVersion: fluxcd.controlplane.io/v1
      kind: ResourceSetInputProvider
      name: my-app-prs
  resourcesTemplate: |
    ---
    apiVersion: kustomize.toolkit.fluxcd.io/v1
    kind: Kustomization
    metadata:
      name: preview-<< inputs.id | slugify >>
      namespace: flux-system
    spec:
      interval: 1m
      sourceRef:
        kind: GitRepository
        name: my-app-repo
      path: ./deploy/kubernetes
      targetNamespace: preview-<< inputs.id | slugify >>
```

This creates preview environments automatically when a PR with the `deploy/preview` label is opened, and tears them down when the PR is closed.

## Comparison with kubara

| kubara Application | kubarax equivalent |
|---|---|
| `Application` CR pointing to a repo | `Kustomization` or `HelmRelease` CR |
| `destination.serverName` | `kubeConfig.secretRef` for remote clusters |
| `repoPath` / `repoUrl` | `sourceRef` + `path` |
| `projectName` scoping | `targetNamespace` + `serviceAccountName` |
| Manual sync / auto-sync | Always auto-synced (GitOps native) |
| App-of-apps pattern | `Kustomization` pointing to a directory of resources |
