# Add External Secrets & ClusterSecretStore

This guide covers how to set up the External Secrets Operator and configure a `ClusterSecretStore` to sync secrets from an external provider (Vault, AWS Secrets Manager, etc.) into your Kubernetes clusters.

## Prerequisites

- External Secrets enabled in `config.yaml`:

```yaml
services:
  externalSecrets:
    status: enabled
```

- A secret backend (Vault, AWS Secrets Manager, GCP Secret Manager, 1Password, etc.)

## Secrets to Pre-Create in Your Secret Manager

The generated ExternalSecret manifests expect certain entries to already exist in your secret backend **before** the corresponding platform services can reconcile. You only need to create secrets for services you have enabled in `config.yaml`.

The default ClusterSecretStore name convention used in the generated templates is `{cluster-name}-{stage}` (e.g. `my-platform-prod`).

### Core Platform Secrets

| Service | Condition | Key | Property | Description |
|---------|-----------|-----|----------|-------------|
| cert-manager | solver = `cloudflare` | `apiKey` | `cloudflare-api-token` | Cloudflare API token for DNS-01 challenge |
| cert-manager | solver = `route53` | `cert-manager` | `access_key_id` | AWS access key for Route53 DNS-01 challenge |
| cert-manager | solver = `route53` | `cert-manager` | `secret_access_key` | AWS secret key for Route53 DNS-01 challenge |
| external-dns | default template uses Cloudflare | `apiKey` | `cloudflare-api-token` | Cloudflare API token for DNS record management |
| oauth2-proxy | SSO enabled | `oauth2-proxy` | `client-id` | OIDC client ID |
| oauth2-proxy | SSO enabled | `oauth2-proxy` | `client-secret` | OIDC client secret |
| oauth2-proxy | SSO enabled | `oauth2-proxy` | `cookie-secret` | Random 32-byte base64 cookie encryption key |
| kube-prometheus-stack | Grafana OAuth enabled | `grafana` | `client-id` | Grafana OIDC client ID |
| kube-prometheus-stack | Grafana OAuth enabled | `grafana` | `client-secret` | Grafana OIDC client secret |

### Multi-Cluster & Repository Secrets

These are only needed if you add worker clusters or additional Git repositories:

| Use Case | Key | Property | Description |
|----------|-----|----------|-------------|
| Worker cluster kubeconfig | `k8s/{worker-name}` | `kubeconfig` | Full kubeconfig YAML for the worker cluster |
| Git repository credentials | `git/{repo-name}` | `username` | Git username |
| Git repository credentials | `git/{repo-name}` | `pat` | Git personal access token |

### Example: 1Password

If using [1Password Connect](https://developer.1password.com/docs/connect/) as your External Secrets provider, create items in your vault matching the key/property pairs above:

| 1Password Vault | Item Name | Field | Example Value |
|-----------------|-----------|-------|---------------|
| `kubarax` | `cert-manager` | `cloudflare-api-token` | `cf-token-...` |
| `kubarax` | `cloudflare` | `api_token` | `cf-token-...` |
| `kubarax` | `oauth2-proxy` | `client-id` | `abc123` |
| `kubarax` | `oauth2-proxy` | `client-secret` | `secret-...` |
| `kubarax` | `oauth2-proxy` | `cookie-secret` | *(base64 random)* |
| `kubarax` | `grafana` | `client-id` | `abc123` |
| `kubarax` | `grafana` | `client-secret` | `secret-...` |

### Example: HashiCorp Vault

```bash
vault kv put secret/cert-manager cloudflare-api-token="cf-token-..."
vault kv put secret/cloudflare api_token="cf-token-..."
vault kv put secret/oauth2-proxy client-id="abc123" client-secret="..." cookie-secret="$(openssl rand -base64 32)"
vault kv put secret/grafana client-id="abc123" client-secret="..."
```

> **Note**: The **key** column maps to the secret path or item name in your provider. The **property** column maps to the field or attribute within that secret. Adjust path prefixes to match your ClusterSecretStore provider configuration.

## Setting Up a ClusterSecretStore

A `ClusterSecretStore` is a cluster-scoped resource that tells the External Secrets Operator how to connect to your secret backend. There are two ways to configure it:

### Option A: During Bootstrap

Use the `--with-es-css-file` flag to apply a ClusterSecretStore manifest during cluster bootstrap. The file supports go-template + sprig syntax with access to your cluster config fields (`.name`, `.stage`, `.dnsName`, `.services`, etc.).

Create a manifest file:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: {{ .name }}-vault
spec:
  provider:
    vault:
      server: "https://vault.example.com"
      path: "secret"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "{{ .name }}-external-secrets"
```

Then bootstrap with:

```bash
kubarax bootstrap my-cluster --with-es --with-es-css-file ./cluster-secret-store.yaml
```

The manifest is rendered with your cluster configuration and applied after the External Secrets Operator is installed.

### Option B: Via GitOps (Recommended for Ongoing Management)

Define ClusterSecretStores declaratively in your customer overlay values. After running `kubarax generate --helm`, edit the generated values file at:

```
customer-service-catalog/helm/<cluster-name>/external-secrets/values.yaml
```

Add your ClusterSecretStore configuration:

```yaml
clusterSecretStores:
  controlplane-vault:
    provider:
      vault:
        server: "https://vault.example.com"
        path: "secret"
        auth:
          kubernetes:
            mountPath: "kubernetes"
            role: "external-secrets"
    refreshInterval: 30

  aws-secrets:
    provider:
      aws:
        service: SecretsManager
        region: eu-central-1
        auth:
          jwt:
            serviceAccountRef:
              name: external-secrets-sa
```

Each key under `clusterSecretStores` becomes a `ClusterSecretStore` resource managed by FluxCD. Changes are automatically reconciled.

## Creating ExternalSecrets

Once a `ClusterSecretStore` exists, create `ExternalSecret` resources to sync individual secrets:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-credentials
  namespace: my-app
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: controlplane-vault
  target:
    name: my-app-credentials
  data:
    - secretKey: username
      remoteRef:
        key: apps/my-app
        property: username
    - secretKey: password
      remoteRef:
        key: apps/my-app
        property: password
```

## Common Use Cases

| Use Case | Guide |
|----------|-------|
| Worker cluster kubeconfigs | [Add Worker Cluster](add-worker-cluster.md) (Option B) |
| OAuth2 Proxy credentials | [Add SSO](add-sso.md) |
| Git repository credentials | [Add Repository](add-repository.md) |
