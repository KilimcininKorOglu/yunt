# Yunt Kubernetes Deployment

This directory contains Kubernetes manifests for deploying Yunt mail server in a Kubernetes cluster.

## Prerequisites

| Requirement                 | Description                                           |
|-----------------------------|-------------------------------------------------------|
| Kubernetes Cluster          | Version 1.25 or higher                                |
| kubectl                     | Configured to access your cluster                     |
| Ingress Controller          | For HTTP routing (nginx-ingress, traefik, etc.)       |
| Storage Class               | For persistent volume provisioning                    |
| (Optional) cert-manager     | For automatic TLS certificate management              |

## Quick Start

### 1. Create Namespace (Optional)

```bash
kubectl create namespace yunt
kubectl config set-context --current --namespace=yunt
```

### 2. Configure Secrets

Edit `secret.yaml` and set secure values for:
- `YUNT_AUTH_JWTSECRET`: JWT signing key (generate with `openssl rand -hex 32`)
- `YUNT_ADMIN_PASSWORD`: Admin user password

```bash
# Generate a secure JWT secret
openssl rand -hex 32
```

### 3. Apply Manifests

```bash
# Apply all manifests at once
kubectl apply -f .

# Or apply individually in order
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f pvc.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service-smtp.yaml
kubectl apply -f service-imap.yaml
kubectl apply -f service-http.yaml
kubectl apply -f ingress.yaml
```

### 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -l app=yunt

# Check services
kubectl get svc -l app=yunt

# Check ingress
kubectl get ingress yunt-ingress

# View logs
kubectl logs -l app=yunt -f
```

## Manifest Files

| File               | Description                                    |
|--------------------|------------------------------------------------|
| `deployment.yaml`  | Deployment with probes and resource limits     |
| `service-smtp.yaml`| Service for SMTP (port 1025)                   |
| `service-imap.yaml`| Service for IMAP (port 1143)                   |
| `service-http.yaml`| Service for Web UI/API (port 8025)             |
| `configmap.yaml`   | Non-sensitive configuration                    |
| `secret.yaml`      | Sensitive configuration (JWT secret, password) |
| `pvc.yaml`         | Persistent volume claim for data storage       |
| `ingress.yaml`     | Ingress for HTTP/HTTPS routing                 |

## Configuration

### Using Environment Variables

The ConfigMap (`configmap.yaml`) provides environment-based configuration:

```yaml
YUNT_SERVER_NAME: "localhost"
YUNT_SERVER_DOMAIN: "localhost"
YUNT_DATABASE_DRIVER: "sqlite"
# ... more settings
```

### Using Configuration File

Alternatively, mount the full configuration file:

1. Edit the `yunt-config-file` ConfigMap in `configmap.yaml`
2. Uncomment the volume mount in `deployment.yaml`:

```yaml
volumeMounts:
  - name: config-file
    mountPath: /etc/yunt
    readOnly: true

volumes:
  - name: config-file
    configMap:
      name: yunt-config-file
```

## Accessing Services

### Web UI (via Ingress)

1. Update `ingress.yaml` with your domain name
2. Configure DNS to point to your Ingress controller's IP
3. Access via browser: `https://mail.example.com`

### SMTP (Internal)

```bash
# Port-forward for local testing
kubectl port-forward svc/yunt-smtp 1025:1025

# Send test email
echo "Test email" | curl smtp://localhost:1025 \
  --mail-from sender@example.com \
  --mail-rcpt recipient@example.com \
  -T -
```

### IMAP (Internal)

```bash
# Port-forward for local testing
kubectl port-forward svc/yunt-imap 1143:1143

# Connect with mail client to localhost:1143
```

## Production Considerations

### High Availability

For PostgreSQL or MySQL backends, you can run multiple replicas:

```yaml
spec:
  replicas: 3
```

Note: SQLite requires single replica due to file locking.

### Resource Tuning

Adjust resources based on your workload:

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi
```

### External Database

For production, use an external database:

1. Deploy PostgreSQL or use managed database service
2. Update ConfigMap:

```yaml
YUNT_DATABASE_DRIVER: "postgres"
YUNT_DATABASE_HOST: "postgres-service"
YUNT_DATABASE_PORT: "5432"
YUNT_DATABASE_NAME: "yunt"
```

3. Add database credentials to Secret:

```yaml
YUNT_DATABASE_USER: "yunt"
YUNT_DATABASE_PASSWORD: "secure-password"
```

### TLS/SSL

For secure connections:

1. Install cert-manager for automatic certificates
2. Uncomment TLS section in `ingress.yaml`
3. For SMTP/IMAP TLS, configure TLS secrets and update ConfigMap

### Network Policies

Restrict pod-to-pod communication:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: yunt-network-policy
spec:
  podSelector:
    matchLabels:
      app: yunt
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - port: 8025
    - ports:
        - port: 1025
        - port: 1143
  egress:
    - {}
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -l app=yunt

# Check logs
kubectl logs -l app=yunt --previous
```

### Health Check Failures

```bash
# Test health endpoint manually
kubectl exec -it deployment/yunt -- curl -s http://localhost:8025/ready
kubectl exec -it deployment/yunt -- curl -s http://localhost:8025/health
```

### Storage Issues

```bash
# Check PVC status
kubectl get pvc yunt-data-pvc

# Check PV binding
kubectl describe pvc yunt-data-pvc
```

### Ingress Not Working

```bash
# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Verify ingress configuration
kubectl describe ingress yunt-ingress
```

## Cleanup

```bash
# Delete all resources
kubectl delete -f .

# Delete namespace (if created)
kubectl delete namespace yunt
```

## Related Documentation

| Resource                    | Link                                                  |
|-----------------------------|-------------------------------------------------------|
| Yunt Documentation          | See main project README                               |
| Kubernetes Documentation    | https://kubernetes.io/docs/                           |
| NGINX Ingress Controller    | https://kubernetes.github.io/ingress-nginx/           |
| cert-manager                | https://cert-manager.io/docs/                         |
