# xKS-to-OCP fixed-identity PostgreSQL

This Kubernetes-source-only role deploys a PostgreSQL StatefulSet with a data
PVC, three deterministic orders, and data plus write/read validation.

It deliberately uses low explicit identities to exercise an xKS-to-OCP target
transform:

- pod `runAsUser: 27` and `runAsGroup: 27`;
- pod `fsGroup: 26`, `supplementalGroups: [27]`, and `fsGroupChangePolicy`;
- container `runAsUser: 26` and `runAsGroup: 26`, overriding the pod values.

It retains `runAsNonRoot`, RuntimeDefault seccomp, no privilege escalation, and
dropped capabilities. Restricted OpenShift SCC rejects the explicit low
identity values. For the target experiment, temporarily remove all listed
identity fields from the rendered StatefulSet before applying it; do not change
the source manifest. The target validation confirms whether the database and
migrated PVC work with SCC-assigned identity values.
