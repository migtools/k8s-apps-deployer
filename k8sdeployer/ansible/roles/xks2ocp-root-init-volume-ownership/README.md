# xKS-to-OCP root init volume ownership

This Kubernetes-source fixture is a PostgreSQL StatefulSet with an RWO PVC,
deterministic orders data, and data plus write/read validation. Before the main
database container starts, `repair-volume-ownership` runs as root and repairs
the PVC ownership and group mode expected by the source image.

The PostgreSQL container is deliberately arbitrary-UID-compatible. The root
init container is the migration constraint: a restricted OpenShift SCC rejects
its `runAsUser: 0` request. Removing the root identity is not a valid transform
because the replacement arbitrary UID cannot perform the required ownership
repair.

`target-prerequisites/custom-scc.yml.j2` is a target-admin artifact. A platform
administrator must review and create this SCC, or an equivalent approved SCC,
before a non-admin migration user applies the preserved workload. It is bound
only to this role's named ServiceAccount and permits only the standard safe
volume types, RuntimeDefault seccomp, no privilege escalation, and the `CHOWN`,
`DAC_OVERRIDE`, and `FOWNER` capabilities needed by the repair init container.

The application role never creates or modifies an SCC. Crane should preserve
this workload's root init security context and report the target SCC as an
explicit administrator prerequisite.
