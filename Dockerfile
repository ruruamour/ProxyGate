ARG RUNTIME_BASE_IMAGE=docker.m.daocloud.io/library/debian:bookworm-slim

# 公共基础阶段：只保留运行时必需依赖，避免 BuildKit 并行跑两次 apt-get update
FROM ${RUNTIME_BASE_IMAGE} AS base
ARG RUNTIME_BASE_IMAGE
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy

ENV DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    sed -i 's|http://deb.debian.org|http://mirrors.ustc.edu.cn|g' /etc/apt/sources.list.d/debian.sources; \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "20";\nAcquire::https::Timeout "20";\n' >/etc/apt/apt.conf.d/80-retries; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates tzdata curl; \
    rm -rf /var/lib/apt/lists/*

# 构建阶段（避免拉取超大的 golang 基础镜像，直接在 Debian 内下载 Go 工具链）
FROM base AS builder
ARG GO_VERSION=1.25.0
ARG GO_DOWNLOAD_BASE=https://golang.google.cn/dl
ARG SINGBOX_VERSION=1.13.5
ARG SINGBOX_DOWNLOAD_BASE=https://ghproxy.net/https://github.com/SagerNet/sing-box/releases/download
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPROXY=https://goproxy.cn|https://goproxy.io|direct
ENV GOSUMDB=off
ENV PATH=/usr/local/go/bin:${PATH}

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends gcc libc6-dev git xz-utils; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "${arch}" in \
      amd64) go_arch="amd64" ;; \
      arm64) go_arch="arm64" ;; \
      *) echo "unsupported architecture: ${arch}" >&2; exit 1 ;; \
    esac; \
    curl --retry 5 --retry-all-errors --connect-timeout 20 --max-time 600 -fsSL \
      "${GO_DOWNLOAD_BASE}/go${GO_VERSION}.linux-${go_arch}.tar.gz" \
      -o /tmp/go.tar.gz; \
    tar -C /usr/local -xzf /tmp/go.tar.gz; \
    rm -f /tmp/go.tar.gz

WORKDIR /app
COPY go.mod go.sum ./
RUN set -eux; \
    for attempt in 1 2 3 4 5; do \
      go mod download && exit 0; \
      echo "go mod download failed on attempt ${attempt}, retrying..." >&2; \
      sleep "$((attempt * 2))"; \
    done; \
    exit 1

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o proxygate .

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "${arch}" in \
      amd64) singbox_arch="amd64" ;; \
      arm64) singbox_arch="arm64" ;; \
      *) echo "unsupported architecture: ${arch}" >&2; exit 1 ;; \
    esac; \
    archive="sing-box-${SINGBOX_VERSION}-linux-${singbox_arch}.tar.gz"; \
    download_ok=""; \
    for url in \
      "${SINGBOX_DOWNLOAD_BASE}/v${SINGBOX_VERSION}/${archive}" \
      "https://ghfast.top/https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/${archive}" \
      "https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/${archive}"; do \
      if curl --retry 2 --retry-all-errors --connect-timeout 20 --max-time 600 -fsSL "${url}" -o /tmp/sing-box.tar.gz; then \
        download_ok="1"; \
        break; \
      fi; \
      rm -f /tmp/sing-box.tar.gz; \
    done; \
    test -n "${download_ok}"; \
    tar -xzf /tmp/sing-box.tar.gz -C /tmp; \
    cp "/tmp/sing-box-${SINGBOX_VERSION}-linux-${singbox_arch}/sing-box" /app/sing-box; \
    chmod +x /app/sing-box; \
    rm -rf /tmp/sing-box*

# 运行阶段：直接复用 base，避免再次 apt-get update
FROM base AS runtime

ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/proxygate .
COPY --from=builder /app/sing-box /usr/local/bin/sing-box

EXPOSE 7776 7777 7778 7779 7780

CMD ["./proxygate"]
