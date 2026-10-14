# Windows MTA-Ops Testing Playbook

This document records the working setup used to run Crane/MTA-Ops end-to-end
tests from a Windows VM while running `k8s-apps-deployer` inside the Linux
Podman machine.

Do not put passwords, bearer tokens, private keys, or complete kubeconfig files
in this document or in the repository. The credentials used during the original
setup were supplied interactively and must be rotated if they are exposed.

## Goal

Run the Windows-native CLI and Ginkgo test process on a Windows VM while using
the Linux-only Ansible controller inside `k8s-apps-deployer`.

The tested architecture is:

```text
Windows VM
  native crane.exe
  native ginkgo.exe
  native oc.exe
  native kubectl.exe
  native k8sdeploy-podman.exe
        |
        | podman run
        v
Podman Linux machine
  quay.io/migqe/k8s-apps-deployer:multiarch
  Python, Ansible, ansible-runner
        |
        v
OpenShift source and target clusters
```

The Windows CLI remains native. Only `k8sdeploy` and its Ansible controller
run inside the Linux container.

## Why A Container Is Required

`k8s-apps-deployer` calls `ansible_runner.run()` from
`k8sdeployer/apps/ansible_app.py`. Ansible controller execution is not natively
supported on Windows because the controller stack depends on POSIX behavior.
Typical native-Windows failures include:

```text
ModuleNotFoundError: No module named 'fcntl'
```

Ansible can manage Windows systems, but that is different from running the
Ansible controller on Windows. WSL2, a Linux container, or a remote Linux
helper is required.

## Windows VM Findings

The test VM was reachable over SSH and reported:

- Windows Server 2025 Datacenter
- x64 architecture
- AWS `c8i.4xlarge`
- Podman Desktop/CLI `6.1.1`
- A Hyper-V Podman machine named `podman-machine-default`

WSL was not installed, and both `Microsoft-Windows-Subsystem-Linux` and
`VirtualMachinePlatform` were disabled. Podman was therefore used instead of
WSL2.

## Podman Machine Warning

The Podman machine API forwarding process is sensitive to how it is started.
Starting or restarting the machine through a non-interactive SSH session can
leave the machine marked as running while its API forwarding socket is gone.
Symptoms include:

```text
Cannot connect to Podman
connectex: No connection could be made because the target machine actively refused it
```

Another failure mode is a stale `gvproxy.exe` process holding the machine SSH
port, commonly `127.0.0.1:49923`.

Operational rules:

- Start or restart the Podman machine from an interactive PowerShell session.
- Do not run `podman machine stop`, `start`, `rm`, or `init` through an SSH
  automation command unless the process lifetime is known to survive the SSH
  session.
- Do not delete the Podman machine merely because `podman run` fails; inspect
  the machine and forwarding process first.
- The wrapper does not start or repair the Podman machine. It assumes Podman is
  already healthy.

Useful interactive checks:

```powershell
podman machine list
podman system connection list
podman info
```

If the machine is healthy, the default connection should point to the
`podman-machine-default` Linux socket.

## Container Image

The image is published as:

```text
quay.io/migqe/k8s-apps-deployer:multiarch
```

The manifest contains:

- `linux/arm64`
- `linux/amd64`

Podman automatically selects `linux/amd64` on the Windows VM.

The image contains:

- Python 3.12
- `k8s-apps-deployer`
- Ansible
- `ansible-runner`
- Kubernetes Python client `35.0.0`
- OpenShift Python client
- Linux `kubectl` `v1.31.4`
- All packaged Ansible roles

The Kubernetes Python client was pinned to `35.0.0` because the unbounded
dependency range could resolve the incompatible `37.x` client.

### Pulling And Verifying The Image

Run this from an interactive PowerShell session on Windows:

```powershell
podman pull quay.io/migqe/k8s-apps-deployer:multiarch
podman run --rm quay.io/migqe/k8s-apps-deployer:multiarch list
podman run --rm --entrypoint kubectl `
  quay.io/migqe/k8s-apps-deployer:multiarch version --client=true
```

The wrapper uses `--pull=missing`. This pulls the image if it is absent but
does not refresh an already cached mutable tag. Pull the image once before a
test run when updating it:

```powershell
podman pull quay.io/migqe/k8s-apps-deployer:multiarch
```

Using `--pull=always` for every deployer invocation is not recommended because
a single test can invoke `k8sdeploy` many times and could receive different
image contents during one run.

### Building The Multi-Architecture Image

The Dockerfile detects the target architecture for `kubectl` and supports an
optional `KUBECTL_ARCH` override. The local multi-architecture manifest was
built with Podman and pushed with:

```bash
podman manifest create localhost/k8s-apps-deployer:multiarch
podman build --platform linux/arm64 \
  --manifest localhost/k8s-apps-deployer:multiarch .
podman build --platform linux/amd64 \
  --manifest localhost/k8s-apps-deployer:multiarch .
podman manifest push --all \
  localhost/k8s-apps-deployer:multiarch \
  docker://quay.io/migqe/k8s-apps-deployer:multiarch
```

The image must be pushed to a registry before Windows can pull it. A local
laptop image is not automatically visible to the Windows VM.

## Windows CLI Installation

The following tools were installed or verified on the Windows VM:

| Tool | Version | Location |
| --- | --- | --- |
| Git | `2.55.0.windows.3` | `C:\Program Files\Git\cmd\git.exe` |
| Go | `1.27.2` | `C:\Program Files\Go\bin\go.exe` |
| Ginkgo | `v2.28.1` | `C:\Users\Administrator\go\bin\ginkgo.exe` |
| OpenShift CLI | `4.22.17` | `C:\Tools\oc.exe` |
| kubectl | `v1.31.4` | `C:\Tools\kubectl.exe` |

`C:\Tools`, the Go binary directory, and the user Go install directory were
added to the machine PATH.

The Ginkgo version matches the repository's `go.mod`:

```text
github.com/onsi/ginkgo/v2 v2.28.1
```

## OpenShift Contexts

The Windows kubeconfig contains four stable contexts:

| Context | Identity | Purpose |
| --- | --- | --- |
| `src-ocp` | `kube:admin` | Source cluster administration |
| `src-ocp-nonadmin` | `dev` | Source namespace-admin test identity |
| `tgt-ocp` | `kube:admin` | Target cluster administration |
| `tgt-ocp-nonadmin` | `dev` | Target namespace-admin test identity |

The admin contexts were created from the supplied source and target
`oc login` commands. The non-admin contexts were created by logging into both
clusters as `dev/dev`; the user already existed, so no OAuth or HTPasswd
configuration was changed.

Validate the contexts without printing tokens or kubeconfig contents:

```powershell
oc whoami --context=src-ocp
oc whoami --context=src-ocp-nonadmin
oc whoami --context=tgt-ocp
oc whoami --context=tgt-ocp-nonadmin
```

The kubeconfig must be available to both native Windows commands and the Linux
container. The wrapper mounts the Windows kubeconfig at `/kube/config` and sets
`KUBECONFIG=/kube/config` inside the container.

For remote OpenShift clusters, the kubeconfig server URLs are reachable from
the Podman Linux machine. A local Minikube kubeconfig using
`127.0.0.1:<port>` is not generally usable from the container because the
container's loopback interface is not the Windows host's loopback interface.

## Wrapper Scripts

The wrappers are now stored in this repository.

### Unix Wrapper

Path:

```text
k8sdeploy-podman.sh
```

Usage:

```bash
./k8sdeploy-podman.sh --context src-ocp --insecure-skip-tls-verify list
```

The default image is the published multi-architecture image. Local development
can override it:

```bash
K8SDEPLOY_IMAGE=k8s-apps-deployer:test \
  ./k8sdeploy-podman.sh --context src-ocp list
```

The kubeconfig path can also be overridden:

```bash
KUBECONFIG=/path/to/config ./k8sdeploy-podman.sh list
```

### Windows Wrapper

Source path:

```text
scripts/windows/k8sdeploy-podman/main.go
```

Build it from its own nested Go module:

```powershell
Set-Location C:\Users\Administrator\k8s-apps-deployer\scripts\windows\k8sdeploy-podman
go build -o C:\Tools\k8sdeploy-podman.exe .
```

The wrapper uses the same `K8SDEPLOY_IMAGE` override as the Unix wrapper:

```powershell
$env:K8SDEPLOY_IMAGE = "quay.io/migqe/k8s-apps-deployer:multiarch"
& C:\Tools\k8sdeploy-podman.exe `
  --context src-ocp `
  --insecure-skip-tls-verify `
  list
```

The native executable is intentional. Ginkgo invokes `--k8sdeploy-bin` using
`exec.Command` and passes an exact argument array. A simple `.cmd` wrapper can
reparse and corrupt JSON arguments such as:

```text
--extra-vars {"non_admin_user":"true","replicaset":{"replicas":2}}
```

A PowerShell or Python script would require an interpreter command in addition
to the script path, which the current `--k8sdeploy-bin` flag does not model.
The Go wrapper produces one native executable, preserves arguments, forwards
standard input/output/error, and returns the child exit code.

## Crane Checkout And Build

Windows cannot safely check out every Crane fixture path because some tracked
fixture names contain characters that are invalid on NTFS. A Windows-safe
archive was used, excluding the invalid OpenShift golden-manifest directory.

From the Windows Crane checkout:

```powershell
Set-Location C:\Users\Administrator\crane
go build -o crane.exe .
crane.exe version
```

## Focused MTA-805 Run

The successful Windows focused test command was:

```powershell
Set-Location C:\Users\Administrator\crane
$env:Path = "C:\Tools;C:\Program Files\Go\bin;C:\Users\Administrator\go\bin;" + $env:Path

ginkgo.exe run -r --timeout=90m -v `
  --focus="\[MTA-805\]" `
  e2e-tests/tests/ -- `
  --k8sdeploy-bin=C:\Tools\k8sdeploy-podman.exe `
  --crane-bin=C:\Users\Administrator\crane\crane.exe `
  --source-context=src-ocp `
  --target-context=tgt-ocp `
  --source-nonadmin-context=src-ocp-nonadmin `
  --target-nonadmin-context=tgt-ocp-nonadmin `
  --verbose-logs `
  --insecure-skip-tls-verify
```

Result:

```text
Ran 1 of 45 Specs
1 Passed
0 Failed
44 Skipped
SUCCESS
```

The test completed in approximately 31 seconds. It created the `sets`
namespace and namespace-scoped RBAC, deployed the DaemonSet, ReplicaSet, and
StatefulSet workload, ran the Crane export/transform/apply pipeline, validated
the target resources, and cleaned up the workloads, rolebindings, and test
namespaces.

This run confirmed that nested JSON extra variables survived the Windows
Ginkgo -> Go wrapper -> Podman -> Linux `k8sdeploy` path.

## Important Test Observations

### Global CLI Flags

`k8sdeploy` global flags must appear before the subcommand:

```text
k8sdeploy -v --context src-ocp deploy simple-nginx-nopv -n test
```

This is incorrect:

```text
k8sdeploy deploy simple-nginx-nopv -n test -v
```

### Basic Pod Role

An initial `basic-pod` smoke attempt returned success but skipped all role work.
The role's task file checks role-specific variables such as
`with_deploy_two_pvs_base_data`, while the generic deploy playbook sets only
`with_deploy`. This is a deployer role behavior issue, not an image or Windows
integration issue.

`simple-nginx-nopv` was used for the real deploy/validate/remove smoke test
because it correctly consumes `with_deploy`, `with_validate`, and
`with_cleanup`.

### Image Pull Policy

The wrappers use `--pull=missing`:

- Missing image: Podman pulls it.
- Existing image: Podman reuses the local copy.

Pull the image once before a test run when a mutable tag has changed. Do not
pull on every `k8sdeploy` invocation because one Ginkgo test may invoke the
deployer many times.

## Cleanup

After an interrupted test, inspect only known test namespaces:

```powershell
oc --context=src-ocp get namespace sets
oc --context=tgt-ocp get namespace sets
```

Delete only test-owned namespaces:

```powershell
oc --context=src-ocp delete namespace sets --ignore-not-found=true --wait=true
oc --context=tgt-ocp delete namespace sets --ignore-not-found=true --wait=true
```

Do not delete shared namespaces, cluster-wide OAuth configuration, or unrelated
workloads.

## Jira And Pull Request

Tracking Jira:

```text
MTA-8024 - Enable MTA-Ops testing on Windows 11
```

Wrapper pull request:

```text
https://github.com/migtools/k8s-apps-deployer/pull/12
```

## Security Notes

- Credentials supplied for the ephemeral clusters must not be committed.
- Rotate any password that has been exposed in chat, shell history, logs, or
  copied files.
- Do not commit kubeconfig contents.
- Prefer short-lived cluster credentials where possible.
- Use `--insecure-skip-tls-verify` only for the explicitly ephemeral test
  clusters that require it.
- Keep the Podman machine and Windows SSH access restricted.
