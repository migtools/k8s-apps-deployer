# xKS-to-OCP custom-SCC fixed-identity database

This Kubernetes-source fixture represents a legacy MySQL workload that must
retain UID, GID, `fsGroup`, and supplemental group `27`. It uses a named
ServiceAccount and one RWO PVC, seeds deterministic order data, and validates
the migrated data plus a new database write/read.

Before MySQL starts, a hardened init container verifies that its effective UID
and GID are both `27`. This models a legacy vendor startup contract that
refuses an unexpected Unix identity. Consequently, removing the explicit
identity fields causes the target Pod to fail during initialization under the
OpenShift-assigned identity; preservation with the approved custom SCC is
required for this fixture.

The source manifest deliberately sets UID/GID/group `27` at both pod and
container scope. Removing these fields is not the supported path for this
fixture. The target workload must preserve them.

`target-prerequisites/custom-scc.yml.j2` is a target-admin artifact. Before
the non-admin migration user applies the rendered workload, a target platform
administrator must review and create that least-privilege SCC or an equivalent
approved SCC. It permits only this role's named ServiceAccount in the target
namespace to use UID/GID/group `27` with PVC and standard projected volumes.

The application role never creates or modifies an SCC. Do not export this SCC
from a generic Kubernetes source and do not apply it automatically during a
non-admin migration.
