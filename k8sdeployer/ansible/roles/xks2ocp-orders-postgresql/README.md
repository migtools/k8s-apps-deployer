# xKS-to-OCP orders PostgreSQL

This role deploys a provider-neutral, data-bearing PostgreSQL orders backend for
xKS-to-OpenShift migration testing. It uses a StatefulSet and a
`volumeClaimTemplate`, seeds three deterministic rows, and validates both the
migrated rows and a target-side database write/read.

The workload is compatible with OpenShift's restricted SCC: it requests
non-root execution, disables privilege escalation, drops all Linux capabilities,
and uses the RuntimeDefault seccomp profile. It intentionally does not specify
`runAsUser`, `runAsGroup`, or supplemental groups, allowing OpenShift to assign
a permitted arbitrary UID. It sets `fsGroup: 0`, which is accepted by the
default restricted SCC and lets the PostgreSQL image write to a newly provisioned
volume through its root group.

Invoke the application through the normal app-deployer role name:

```text
xks2ocp-orders-postgresql
```

This is the durable database tier only; a separate HTTP API layer can be added
later when API-to-database migration validation is needed.
