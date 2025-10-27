---
command: kubectl
aliases: [k8s, kube, kubernetes]
keywords: [pods, deployments, services, namespace, container, cluster, logs, describe, exec, scale, delete, apply]
categories: [devops, container-orchestration, kubernetes]
priority: high
version: "1.28"
---

# kubectl - Kubernetes Command Line Tool

## Overview

kubectl is the Kubernetes command-line tool for managing K8s clusters. It allows you to run commands against Kubernetes clusters to deploy applications, inspect and manage cluster resources, and view logs.

## Common Patterns

### Viewing Resources

```bash
# List all pods in all namespaces
kubectl get pods -A

# List pods in specific namespace
kubectl get pods -n <namespace>

# List with more details
kubectl get pods -o wide

# List deployments
kubectl get deployments

# List services
kubectl get services
kubectl get svc
```

### Describing Resources

```bash
# Describe a pod
kubectl describe pod <pod-name> -n <namespace>

# Describe a deployment
kubectl describe deployment <name> -n <namespace>

# Describe a service
kubectl describe svc <name> -n <namespace>
```

### Logs

```bash
# View logs
kubectl logs <pod-name> -n <namespace>

# Follow logs (tail -f)
kubectl logs -f <pod-name> -n <namespace>

# View previous container logs
kubectl logs --previous <pod-name> -n <namespace>

# Logs from all pods with a label
kubectl logs -f -l app=myapp -n <namespace>
```

### Executing Commands

```bash
# Execute command in pod
kubectl exec <pod-name> -n <namespace> -- <command>

# Interactive shell
kubectl exec -it <pod-name> -n <namespace> -- /bin/bash
kubectl exec -it <pod-name> -n <namespace> -- /bin/sh
```

### Scaling

```bash
# Scale deployment
kubectl scale deployment <name> --replicas=<count> -n <namespace>

# Scale statefulset
kubectl scale statefulset <name> --replicas=<count> -n <namespace>
```

### Applying/Deleting

```bash
# Apply configuration
kubectl apply -f <file.yaml>
kubectl apply -f <directory>/

# Delete resource
kubectl delete pod <name> -n <namespace>
kubectl delete deployment <name> -n <namespace>
```

## Examples

User: "show me all pods"
Command: kubectl get pods -A

User: "list pods in production"
Command: kubectl get pods -n production

User: "follow logs for api service"
Command: kubectl logs -f -l app=api -n production

User: "describe the database pod"
Command: kubectl describe pod -l app=database -n production

User: "scale web deployment to 5"
Command: kubectl scale deployment web --replicas=5 -n production

User: "get into the api container"
Command: kubectl exec -it -l app=api -n production -- /bin/bash

User: "show me production services"
Command: kubectl get svc -n production

User: "what's running in staging"
Command: kubectl get pods -n staging

User: "tail logs from worker pods"
Command: kubectl logs -f -l app=worker -n production

User: "restart the api deployment"
Command: kubectl rollout restart deployment api -n production

User: "check deployment status"
Command: kubectl rollout status deployment api -n production

User: "show recent events"
Command: kubectl get events --sort-by='.lastTimestamp' -n production

User: "delete the old job"
Command: kubectl delete job old-job -n production

User: "show pod resource usage"
Command: kubectl top pods -n production

User: "list all namespaces"
Command: kubectl get namespaces

User: "show all resources in production"
Command: kubectl get all -n production

## Tips

- **Use `-A` or `--all-namespaces`** to search across all namespaces
- **Use `-n <namespace>`** to specify a specific namespace
- **Use `-l <label>`** for label selectors: `kubectl get pods -l app=myapp`
- **Use `-o wide`** for more details in output
- **Use `-o yaml`** or `-o json`** for full resource specs
- **Use `--dry-run=client -o yaml`** to preview resources before applying
- **Use `-f`** to follow logs in real-time
- **Use `--previous`** to see logs from crashed containers
- **Use `watch`** for continuous updates: `watch kubectl get pods -n production`

## Common Namespaces

- `production` - Production environment
- `staging` - Staging environment
- `development` - Development environment
- `kube-system` - Kubernetes system components
- `default` - Default namespace

## Context and Config

```bash
# Show current context
kubectl config current-context

# List all contexts
kubectl config get-contexts

# Switch context
kubectl config use-context <context-name>

# Set default namespace for context
kubectl config set-context --current --namespace=<namespace>
```

## Troubleshooting

```bash
# Check why pod is not running
kubectl describe pod <pod-name> -n <namespace>
kubectl logs <pod-name> -n <namespace>

# Check pod events
kubectl get events --field-selector involvedObject.name=<pod-name> -n <namespace>

# Port forward for local testing
kubectl port-forward pod/<pod-name> 8080:80 -n <namespace>

# Copy files to/from pod
kubectl cp <pod-name>:/path/to/file ./local-file -n <namespace>
kubectl cp ./local-file <pod-name>:/path/to/file -n <namespace>
```

## Safety Notes

- Always specify namespace with `-n` to avoid mistakes
- Use `--dry-run=client` to preview changes
- Be careful with `kubectl delete` commands
- Use RBAC to limit permissions in production
