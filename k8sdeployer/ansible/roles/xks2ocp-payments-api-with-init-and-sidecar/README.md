# xKS-to-OCP payments API with init container and sidecar

This role supplies the successful restricted-SCC counterpart to the
`xks2ocp-root-init-volume-ownership` prerequisite fixture. It deploys:

- a PostgreSQL StatefulSet with a PVC containing deterministic payment data;
- a non-root payments API Deployment;
- a non-root bootstrap init container which validates its mounted API content;
  and
- a non-root log sidecar sharing an `emptyDir` runtime volume with the API.

Every PodTemplate position has its own restrictive security context: the pod,
init container, API container, sidecar, and PostgreSQL container all disable
privilege escalation, drop all capabilities, and use RuntimeDefault seccomp.
There are deliberately no fixed UID/GID, root init container, host access, or
custom-SCC prerequisite fields. The generic-Kubernetes PostgreSQL source does
set `fsGroup: 0`, because a new Azure Disk is root-owned and the image writes
through group `0`; this field is omitted for direct OCP deployment and must be
removed by the xKS-to-OCP transform before target admission.

The fixture intentionally does not write the shared `emptyDir`: a portable,
non-root workload cannot assume a writable group on every source volume without
adding an explicit `fsGroup`, which would obscure the restricted-SCC success
case this role is meant to cover.

The role seeds three payments. Validation checks that the init container
completed, both application containers are ready, the API health endpoint is
reachable, the persisted payment count/digest survives the migration, and the
database accepts a target-side write/read transaction.

Use the application ID:

```text
xks2ocp-payments-api-with-init-and-sidecar
```
