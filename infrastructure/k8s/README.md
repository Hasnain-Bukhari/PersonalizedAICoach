# Kubernetes deployment

The base is intentionally environment-neutral. Before applying it:

1. Replace the base-only `coach-api:local` and `coach-worker:local` image names with immutable registry digests using a Kustomize environment overlay. The base is not a deployable production release until this substitution is present.
2. Create `coach-runtime` through External Secrets, mapping the Terraform-created Secrets Manager value. `secret.template.yaml` documents the current keys without containing usable credentials.
3. Patch `AUTH0_ISSUER`, `AUTH0_AUDIENCE`, `OBJECT_BUCKET`, `AWS_REGION`, and both `000000000000/replace-me-*` service-account role annotations in an environment overlay. Startup must be rejected by delivery policy while any `replace-me` value remains.
4. Install metrics-server, NVIDIA device plugin, AWS Load Balancer Controller, External Secrets, an OpenTelemetry collector, and Karpenter or another node autoscaler.
5. Add an environment Ingress with TLS 1.3 policy and associate the Terraform WAF ACL. The base deliberately exposes no public load balancer.

Render without mutating the cluster using `kubectl kustomize infrastructure/k8s/base`.
