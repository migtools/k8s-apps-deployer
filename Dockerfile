# k8s-apps-deployer container image

FROM python:3.12-slim

ARG KUBECTL_VERSION=v1.31.4
ARG KUBECTL_ARCH

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    KUBECONFIG=/kube/config

WORKDIR /app

# Download kubectl for the target architecture without requiring a package manager.
RUN if [ -z "${KUBECTL_ARCH}" ]; then \
      case "$(uname -m)" in \
        aarch64) KUBECTL_ARCH=arm64 ;; \
        x86_64) KUBECTL_ARCH=amd64 ;; \
        *) echo "Unsupported build architecture: $(uname -m)" >&2; exit 1 ;; \
      esac; \
    fi \
    && python -c "import urllib.request; urllib.request.urlretrieve('https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${KUBECTL_ARCH}/kubectl', '/usr/local/bin/kubectl')" \
    && chmod 0755 /usr/local/bin/kubectl \
    && kubectl version --client=true

COPY pyproject.toml README.md ./
COPY k8sdeployer ./k8sdeployer/

RUN pip install --no-cache-dir .

RUN mkdir -p /kube && chmod 755 /kube

ENTRYPOINT ["k8sdeploy"]
CMD ["--help"]
