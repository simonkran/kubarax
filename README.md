# kubarax

A framework and bootstrapping tool for building and operating a production-grade Kubernetes platform with the **Flux Operator**.

Inspired by [kubara](https://github.com/kubara-io/kubara) (which uses ArgoCD), kubarax translates the same platform engineering concepts to a FluxCD-native approach using the [Flux Operator](https://fluxoperator.dev/).

## Architecture

kubarax follows a **hub-and-spoke** model:

- **Control plane cluster (hub)**: Runs the Flux Operator, manages all platform components and spoke clusters
- **Worker clusters (spokes)**: Managed remotely via FluxCD's `spec.kubeConfig.secretRef` through ResourceSets

### Flux Operator Approach

Instead of installing FluxCD controllers directly, kubarax uses the **Flux Operator** which provides:

- **FluxInstance CR**: Declaratively manages the Flux controller lifecycle, distribution, and sync configuration
- **ResourceSet CR**: Templated deployments with `<< inputs.name >>` syntax for multi-cluster patterns
- **ResourceSetInputProvider CR**: Dynamic input discovery from GitHub PRs, branches, OCI tags, etc.
- **Built-in Web UI**: Lightweight dashboard for tracking workloads and reconciliation status (port 9080)
- **Automated upgrades**: Semver-based version ranges (e.g., `2.x`) for automatic Flux updates

### Mapping from kubara (ArgoCD)

| kubara (ArgoCD) | kubarax (Flux Operator) |
|---|---|
| `Application` | `HelmRelease` + `Kustomization` |
| `ApplicationSet` (cluster generator) | `ResourceSet` + `ResourceSetInputProvider` |
| `AppProject` | Namespace RBAC + ServiceAccount |
| ArgoCD cluster secrets | `spec.kubeConfig.secretRef` on resources |
| ArgoCD repo credentials | FluxInstance `sync.pullSecret` |
| ArgoCD bootstrap | `flux-operator` Helm chart + `FluxInstance` CR |
| Sync waves | `dependsOn` chains |
| ArgoCD UI | Flux Operator Web UI |

## Quick Start

### 1. Initialize

```bash
# Generate .gitignore and example .env
kubarax init --prep

# Fill in the .env file with your values
vim .env

# Generate config.yaml from environment
kubarax init
```

### 2. Generate

```bash
# Generate all Flux Operator manifests (Helm charts + FluxInstance + ResourceSets)
kubarax generate --helm

# Or preview without writing
kubarax generate --helm --dry-run
```

### 3. Bootstrap

```bash
# Bootstrap the Flux Operator onto your control plane cluster
kubarax bootstrap my-cluster --with-es-crds --with-prometheus-crds

# Optionally apply a ClusterSecretStore during bootstrap
kubarax bootstrap my-cluster --with-es-crds --with-es-css-file ./cluster-secret-store.yaml
```

This will:
1. Install the Flux Operator via its OCI Helm chart
2. Create a `FluxInstance` CR that installs and manages all Flux controllers
3. Configure sync from your Git repository
4. Optionally install external-secrets and prometheus CRDs
5. Optionally apply a ClusterSecretStore manifest (supports go-template + sprig)

## Commands

| Command | Description |
|---|---|
| `kubarax init --prep` | Generate `.gitignore` and example `.env` |
| `kubarax init` | Create `config.yaml` from environment variables |
| `kubarax generate` | Generate Flux Operator manifests from templates |
| `kubarax bootstrap <cluster>` | Bootstrap the Flux Operator onto a cluster |
| `kubarax schema` | Generate JSON schema for `config.yaml` validation |

## Generated Structure

```
managed-service-catalog/
  helm/
    flux-operator/           # Flux Operator Helm chart wrapper
    cert-manager/            # Upstream Helm chart wrappers
    traefik/
    external-secrets/
    ...

customer-service-catalog/
  helm/
    <cluster-name>/
      flux-operator/
        fluxinstance.yaml          # FluxInstance CR (manages Flux lifecycle + sync)
        resourceset-platform.yaml  # ResourceSet for platform Kustomization
        resourceset-worker-clusters.yaml  # ResourceSet for worker cluster deployments
      flux-system/
        kustomization.yaml         # FluxCD Kustomization for HelmReleases
        helmrepositories.yaml      # Upstream Helm repos
      helmreleases/
        cert-manager.yaml          # HelmRelease per service with dependsOn chains
        traefik.yaml
        ...
      kustomization.yaml           # Kustomize aggregator
```

## Platform Components

- **Ingress**: Traefik
- **Certificates**: cert-manager
- **DNS**: ExternalDNS
- **Secrets**: External Secrets Operator
- **Monitoring**: kube-prometheus-stack, Metrics Server
- **Logging**: Grafana Loki
- **Policy**: Kyverno
- **Auth**: OAuth2 Proxy
- **Storage**: Longhorn
- **Load Balancer**: MetalLB
- **GitOps UI**: Flux Operator Web UI (built-in)
- **Dashboard**: Homer
- **Git Service**: Forgejo

## Configuration

See `config.yaml` structure:

```yaml
clusters:
  - name: my-platform
    stage: prod
    type: controlplane
    dnsName: platform.example.com
    fluxcd:
      distribution:
        version: "2.x"
        registry: ghcr.io/fluxcd
      cluster:
        type: kubernetes
        size: medium            # small|medium|large resource profiles
        networkPolicy: true
        multitenant: false
      sync:
        kind: GitRepository
        url: https://github.com/org/platform-repo
        ref: refs/heads/main
        path: clusters/my-platform
        interval: 5m
      webUI:
        enabled: true
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      # ... more services
```

## Multi-Cluster with ResourceSets

kubarax generates `ResourceSet` CRs for multi-cluster deployments:

- **Platform ResourceSet**: Deploys the platform Kustomization for the control plane
- **Worker Clusters ResourceSet**: Template for deploying to worker clusters via `kubeConfig.secretRef`

To add worker clusters, either:
1. Add static inputs to the worker-clusters ResourceSet
2. Create a `ResourceSetInputProvider` for dynamic discovery

## Managing Your Platform

- [Add Worker Cluster](docs/add-worker-cluster.md) — Onboard spoke clusters managed from the control plane
- [Add Project](docs/add-project.md) — Create tenant namespaces with RBAC isolation
- [Add Repository](docs/add-repository.md) — Register Git and Helm repositories as Flux sources
- [Add HelmRelease](docs/add-helmrelease.md) — Deploy services via HelmRelease and multi-cluster ResourceSets
- [Add Application](docs/add-application.md) — Deploy standalone applications from Git or Helm
- [Add SSO](docs/add-sso.md) — Configure Single Sign-On with OAuth2 Proxy and OIDC providers

## Building

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
```

## License

Apache 2.0
