# xKS-to-OCP orders PostgreSQL

This role deploys a data-bearing PostgreSQL orders backend on a Kubernetes
source cluster for xKS-to-OpenShift migration testing. It uses a StatefulSet and a
`volumeClaimTemplate`, seeds three deterministic rows, and validates both the
migrated rows and a target-side database write/read.

The source workload requests non-root execution, disables privilege escalation,
drops all Linux capabilities, and uses the RuntimeDefault seccomp profile. It
intentionally sets `fsGroup: 0` and `fsGroupChangePolicy: OnRootMismatch` so the
PostgreSQL image can write to a newly provisioned Kubernetes volume through its
root group.

This is intentionally not an OpenShift-source-compatible application. OpenShift
restricted SCC rejects `fsGroup: 0`. It is a regression fixture for xKS-to-OCP
migration: the target transform must remove incompatible explicit identity
fields, allowing the destination namespace's SCC to assign its permitted values.

Invoke the application through the normal app-deployer role name:

```text
xks2ocp-orders-postgresql
```

This is the durable database tier only; a separate HTTP API layer can be added
later when API-to-database migration validation is needed.
