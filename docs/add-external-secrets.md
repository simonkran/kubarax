# Add External Secrets & ClusterSecretStore

This guide covers how to set up the External Secrets Operator and configure a `ClusterSecretStore` to sync secrets from 1Password (via the 1Password SDK provider) into your Kubernetes clusters.

## Prerequisites

- External Secrets enabled in `config.yaml`:

```yaml
services:
  externalSecrets:
    status: enabled
```

- A 1Password account with a [service account token](https://developer.1password.com/docs/service-accounts/)
- If your provider requires a Kubernetes Secret for authentication (e.g., 1Password service account token), configure it via `.env` — see [Authentication Secret](#authentication-secret) below

## Secrets to Pre-Create in Your Secret Manager

The generated ExternalSecret manifests expect certain entries to already exist in your secret backend **before** the corresponding platform services can reconcile. You only need to create secrets for services you have enabled in `config.yaml`.

The default ClusterSecretStore name convention used in the generated templates is `{cluster-name}-{stage}` (e.g. `my-platform-prod`).

The 1Password vault name is configured on the ClusterSecretStore via the `vault` field in the `onepasswordSDK` provider spec. The **Key** column in the tables below maps to the item name in that vault.

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

### Example: 1Password SDK

Create items in your 1Password vault matching the key/property pairs above:

| 1Password Vault | Item Name | Field | Example Value |
|-----------------|-----------|-------|---------------|
| `kubarax` | `cert-manager` | `cloudflare-api-token` | `cf-token-...` |
| `kubarax` | `cloudflare` | `api_token` | `cf-token-...` |
| `kubarax` | `oauth2-proxy` | `client-id` | `abc123` |
| `kubarax` | `oauth2-proxy` | `client-secret` | `secret-...` |
| `kubarax` | `oauth2-proxy` | `cookie-secret` | *(base64 random)* |
| `kubarax` | `grafana` | `client-id` | `abc123` |
| `kubarax` | `grafana` | `client-secret` | `secret-...` |

> **Note**: The **key** column maps to the item name in your 1Password vault. The **property** column maps to the field within that item. The vault itself is configured on the ClusterSecretStore, so `remoteRef.key` in ExternalSecrets only needs the item name.

## Authentication Secret

The 1Password SDK provider requires a Kubernetes Secret containing your service account token. Kubarax creates it automatically during bootstrap. Add this to your `.env` file:

```bash
KUBARAX_ESS_TOKEN=ops_your-service-account-token
# Optional: customize secret name and key (defaults shown)
KUBARAX_ESS_SECRET_NAME=eso-auth
KUBARAX_ESS_TOKEN_KEY=token
```

This creates an Opaque secret in the `external-secrets` namespace before the ClusterSecretStore is applied. Reference it in your ClusterSecretStore manifest:

```yaml
spec:
  provider:
    onepasswordSDK:
      vault: my-vault
      auth:
        serviceAccountSecretRef:
          name: eso-auth
          key: token
          namespace: external-secrets
```

If `KUBARAX_ESS_TOKEN` is not set, no secret is created.

## Setting Up a ClusterSecretStore

A `ClusterSecretStore` is a cluster-scoped resource that tells the External Secrets Operator how to connect to your secret backend. There are two ways to configure it:

### Option A: During Bootstrap

Use the `--with-es-css-file` flag to apply a ClusterSecretStore manifest during cluster bootstrap. The file supports go-template + sprig syntax with access to your cluster config fields (`.name`, `.stage`, `.dnsName`, `.services`, etc.).

Create a manifest file:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: {{ .name }}-{{ .stage }}
spec:
  provider:
    onepasswordSDK:
      vault: {{ .name }}-{{ .stage }}
      auth:
        serviceAccountSecretRef:
          name: eso-auth
          key: token
          namespace: external-secrets
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
  controlplane-1password:
    provider:
      onepasswordSDK:
        vault: my-vault
        auth:
          serviceAccountSecretRef:
            name: eso-auth
            key: token
            namespace: external-secrets
    refreshInterval: 30
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
    name: controlplane-1password
  target:
    name: my-app-credentials
  data:
    - secretKey: username
      remoteRef:
        key: my-app
        property: username
    - secretKey: password
      remoteRef:
        key: my-app
        property: password
```

## Common Use Cases

| Use Case | Guide |
|----------|-------|
| Worker cluster kubeconfigs | [Add Worker Cluster](add-worker-cluster.md) (Option B) |
| OAuth2 Proxy credentials | [Add SSO](add-sso.md) |
| Git repository credentials | [Add Repository](add-repository.md) |
