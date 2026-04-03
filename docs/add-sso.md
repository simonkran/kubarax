# Add SSO

This guide covers configuring Single Sign-On (SSO) for platform services in kubarax. This is the equivalent of kubara's "Add SSO" and "SSO Examples" workflows.

## Overview

In kubarax, SSO is configured through **OAuth2 Proxy** as a central authentication gateway, combined with OIDC-capable services. Unlike kubara (which configured ArgoCD Dex directly), kubarax uses:

- **OAuth2 Proxy**: Central SSO gateway for all platform services
- **Flux Operator Web UI**: Supports OIDC natively (no Dex/Weave GitOps needed)
- **Grafana**: Built-in OIDC/OAuth support via kube-prometheus-stack values
- **Kyverno Policy Reporter**: Can be placed behind OAuth2 Proxy

## Prerequisites

- A running control plane cluster with platform services deployed
- An OIDC provider (e.g., GitHub, GitLab, Keycloak, Azure AD, Google, Okta)
- DNS configured for your platform domain

## Step 1: Register an OAuth Application

Register an OAuth/OIDC application with your identity provider.

### GitHub OAuth App

1. Go to **Settings → Developer settings → OAuth Apps → New OAuth App**
2. Set the following:
   - **Application name**: `kubarax-platform`
   - **Homepage URL**: `https://dashboard.<your-domain>`
   - **Authorization callback URL**: `https://oauth2.<your-domain>/oauth2/callback`
3. Note the **Client ID** and generate a **Client Secret**

### GitLab OAuth App

1. Go to **Admin Area → Applications → New Application**
2. Set scopes: `openid`, `profile`, `email`
3. Set redirect URI: `https://oauth2.<your-domain>/oauth2/callback`

### Keycloak

1. Create a new **Realm** (e.g., `kubarax`)
2. Create a new **Client** with:
   - **Client ID**: `kubarax-platform`
   - **Root URL**: `https://dashboard.<your-domain>`
   - **Valid Redirect URIs**: `https://oauth2.<your-domain>/oauth2/callback`
3. Enable **Client authentication** and note the client secret

### Azure AD (Entra ID)

1. Go to **App registrations → New registration**
2. Set redirect URI: `https://oauth2.<your-domain>/oauth2/callback` (Web)
3. Create a client secret under **Certificates & secrets**
4. Note the **Application (client) ID** and **Directory (tenant) ID**

## Step 2: Create OAuth2 Proxy Secrets

Store your OIDC credentials using External Secrets or direct Kubernetes secrets.

### Using External Secrets (recommended)

Create an `ExternalSecret` that syncs from your secret store:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: oauth2-proxy-credentials
  namespace: oauth2-proxy
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: platform-secret-store
    kind: ClusterSecretStore
  target:
    name: oauth2-proxy-credentials
  data:
    - secretKey: client-id
      remoteRef:
        key: platform/oauth2-proxy
        property: client-id
    - secretKey: client-secret
      remoteRef:
        key: platform/oauth2-proxy
        property: client-secret
    - secretKey: cookie-secret
      remoteRef:
        key: platform/oauth2-proxy
        property: cookie-secret
```

### Using a direct Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-proxy-credentials
  namespace: oauth2-proxy
type: Opaque
stringData:
  client-id: "<your-client-id>"
  client-secret: "<your-client-secret>"
  cookie-secret: "<generate-with-openssl-rand-base64-32>"
```

Generate a cookie secret:

```bash
openssl rand -base64 32
```

## Step 3: Configure OAuth2 Proxy HelmRelease

Enable the OAuth2 Proxy service in `config.yaml`:

```yaml
clusters:
  - name: my-platform
    services:
      oauth2Proxy:
        status: enabled
```

Then customize the generated HelmRelease values in `customer-service-catalog/helm/<cluster>/helmreleases/oauth2-proxy.yaml`:

### GitHub Example

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: oauth2-proxy
  namespace: flux-system
spec:
  targetNamespace: oauth2-proxy
  chart:
    spec:
      chart: oauth2-proxy
      sourceRef:
        kind: HelmRepository
        name: oauth2-proxy
  values:
    config:
      existingSecret: oauth2-proxy-credentials
    extraArgs:
      provider: github
      email-domain: "*"
      github-org: "your-github-org"          # Restrict to org members
      cookie-domain: ".your-domain.com"
      whitelist-domain: ".your-domain.com"
      scope: "user:email read:org"
    ingress:
      enabled: true
      hosts:
        - oauth2.your-domain.com
      tls:
        - secretName: oauth2-proxy-tls
          hosts:
            - oauth2.your-domain.com
```

### Keycloak / Generic OIDC Example

```yaml
  values:
    config:
      existingSecret: oauth2-proxy-credentials
    extraArgs:
      provider: oidc
      oidc-issuer-url: "https://keycloak.your-domain.com/realms/kubarax"
      email-domain: "*"
      cookie-domain: ".your-domain.com"
      whitelist-domain: ".your-domain.com"
      scope: "openid profile email"
```

### Azure AD Example

```yaml
  values:
    config:
      existingSecret: oauth2-proxy-credentials
    extraArgs:
      provider: oidc
      oidc-issuer-url: "https://login.microsoftonline.com/<tenant-id>/v2.0"
      email-domain: "your-company.com"
      cookie-domain: ".your-domain.com"
      whitelist-domain: ".your-domain.com"
      scope: "openid profile email"
```

## Step 4: Protect Services with OAuth2 Proxy

Add Traefik `IngressRoute` middleware annotations to services you want to protect.

### Traefik ForwardAuth Middleware

Create a `Middleware` resource:

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: oauth2-proxy-auth
  namespace: oauth2-proxy
spec:
  forwardAuth:
    address: http://oauth2-proxy.oauth2-proxy.svc.cluster.local:4180/oauth2/auth
    trustForwardHeader: true
    authResponseHeaders:
      - X-Auth-Request-User
      - X-Auth-Request-Email
      - X-Auth-Request-Groups
```

### Apply to IngressRoutes

Reference the middleware in any Traefik `IngressRoute`:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: my-service
  namespace: my-namespace
spec:
  routes:
    - match: Host(`my-service.your-domain.com`)
      kind: Rule
      middlewares:
        - name: oauth2-proxy-auth
          namespace: oauth2-proxy
      services:
        - name: my-service
          port: 80
```

## Step 5: Configure Grafana SSO

Grafana (via kube-prometheus-stack) supports OIDC natively without OAuth2 Proxy.

Add to your kube-prometheus-stack HelmRelease values:

### GitHub OAuth for Grafana

```yaml
  values:
    grafana:
      grafana.ini:
        server:
          root_url: https://grafana.your-domain.com
        auth.github:
          enabled: true
          allow_sign_up: true
          scopes: user:email,read:org
          auth_url: https://github.com/login/oauth/authorize
          token_url: https://github.com/login/oauth/access_token
          api_url: https://api.github.com/user
          allowed_organizations: your-github-org
          client_id: $__file{/etc/secrets/github-oauth/client-id}
          client_secret: $__file{/etc/secrets/github-oauth/client-secret}
      extraSecretMounts:
        - name: github-oauth
          secretName: grafana-github-oauth
          defaultMode: 0440
          mountPath: /etc/secrets/github-oauth
          readOnly: true
```

### Generic OIDC for Grafana

```yaml
  values:
    grafana:
      grafana.ini:
        server:
          root_url: https://grafana.your-domain.com
        auth.generic_oauth:
          enabled: true
          name: SSO
          allow_sign_up: true
          scopes: openid profile email
          auth_url: https://keycloak.your-domain.com/realms/kubarax/protocol/openid-connect/auth
          token_url: https://keycloak.your-domain.com/realms/kubarax/protocol/openid-connect/token
          api_url: https://keycloak.your-domain.com/realms/kubarax/protocol/openid-connect/userinfo
          client_id: grafana
          client_secret: $__file{/etc/secrets/oidc/client-secret}
      extraSecretMounts:
        - name: oidc-secret
          secretName: grafana-oidc
          defaultMode: 0440
          mountPath: /etc/secrets/oidc
          readOnly: true
```

## Step 6: Flux Operator Web UI Access

The Flux Operator Web UI runs on port 9080 within the cluster. Access it via:

### Port Forward (development)

```bash
kubectl port-forward svc/flux-operator -n flux-system 9080:9080
```

### Expose via Traefik with SSO

Create an `IngressRoute` protected by OAuth2 Proxy:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: flux-webui
  namespace: flux-system
spec:
  routes:
    - match: Host(`flux.your-domain.com`)
      kind: Rule
      middlewares:
        - name: oauth2-proxy-auth
          namespace: oauth2-proxy
      services:
        - name: flux-operator
          port: 9080
```

## Step 7: Commit and Apply

```bash
# Commit the SSO configuration
git add customer-service-catalog/
git commit -m "feat: configure SSO for platform services"
git push

# FluxCD will automatically reconcile the changes
# Verify OAuth2 Proxy is running
kubectl get pods -n oauth2-proxy

# Test the SSO flow
# Navigate to https://oauth2.your-domain.com and verify redirect to your IdP
```

## Troubleshooting

| Issue | Solution |
|---|---|
| Redirect URI mismatch | Ensure the callback URL in your IdP matches `https://oauth2.<domain>/oauth2/callback` exactly |
| Cookie errors | Verify `cookie-domain` matches your platform domain and cookie secret is set |
| 403 after login | Check `github-org`, `email-domain`, or group restrictions in OAuth2 Proxy args |
| Grafana login loop | Ensure `root_url` in `grafana.ini` matches the actual Grafana URL |
| OIDC discovery fails | Verify `oidc-issuer-url` is reachable from inside the cluster |
