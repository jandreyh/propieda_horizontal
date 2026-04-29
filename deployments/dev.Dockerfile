# syntax=docker/dockerfile:1.7

# =============================================================================
# Imagen de desarrollo "everything-in-one" para SaaS Propiedad Horizontal.
#
# Contiene:
#   - Go 1.26 + GOPATH en /go (toolchain pinneado)
#   - Node 22 + pnpm + npm (para apps/web Next.js y apps/mobile Expo)
#   - golangci-lint v2, sqlc, golang-migrate, goimports, lefthook, gopls
#   - PostgreSQL client (psql, pg_dump)
#   - Docker CLI + Compose plugin (para Testcontainers via socket del host)
#   - GitHub CLI (gh), git, ssh
#   - Claude Code (@anthropic-ai/claude-code) global
#   - Expo CLI global
#   - ripgrep, fd-find, jq, vim, nano, tmux, htop
#
# Build:
#   docker compose -f deployments/docker-compose.dev.yml build dev
#
# Uso (con docker-compose.dev.yml):
#   docker compose -f deployments/docker-compose.dev.yml up -d
#   docker compose -f deployments/docker-compose.dev.yml exec dev bash
#   # Dentro del contenedor:
#   claude               # iniciar Claude Code (auth la primera vez)
#   go test ./apps/api/...
#   pnpm --filter web dev
# =============================================================================

FROM debian:bookworm-slim

ARG GO_VERSION=1.26.2
ARG NODE_MAJOR=22
ARG GOLANGCI_LINT_VERSION=v2.11.4
ARG SQLC_VERSION=v1.31.1
ARG MIGRATE_VERSION=v4.19.1
ARG LEFTHOOK_VERSION=v1.13.6
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    TZ=America/Bogota \
    GOPATH=/go \
    GOROOT=/usr/local/go \
    GOCACHE=/root/.cache/go-build \
    GOMODCACHE=/go/pkg/mod \
    PNPM_HOME=/root/.local/share/pnpm \
    PATH=/go/bin:/usr/local/go/bin:/root/.local/share/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# ---------------------------------------------------------------------------
# 1) Paquetes base, locale, timezone
# ---------------------------------------------------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl wget gnupg lsb-release \
        git ssh openssh-client \
        build-essential pkg-config make \
        postgresql-client \
        jq ripgrep fd-find less vim nano htop tmux unzip zip \
        locales tzdata sudo \
        python3 python3-pip \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone \
    && ln -s /usr/bin/fdfind /usr/local/bin/fd \
    && rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# 2) Go (binario oficial — debian no trae 1.26 aun)
# ---------------------------------------------------------------------------
RUN ARCH="${TARGETARCH:-amd64}" \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" \
       | tar -xz -C /usr/local \
    && mkdir -p /go/bin /go/src /go/pkg/mod /root/.cache/go-build \
    && go version

# ---------------------------------------------------------------------------
# 3) Node 22 LTS + pnpm via corepack
#    Nota: NO actualizamos npm con `npm install -g npm@latest` porque el
#    paquete de NodeSource trae un npm con dependencias faltantes (bug conocido,
#    `promise-retry` MODULE_NOT_FOUND). El npm bundleado de Node 22 LTS sirve.
# ---------------------------------------------------------------------------
RUN curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && corepack enable \
    && corepack prepare pnpm@latest --activate \
    && node --version && npm --version && pnpm --version

# ---------------------------------------------------------------------------
# 4) Docker CLI + Compose plugin (sin daemon — usa el socket del host)
# ---------------------------------------------------------------------------
RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg \
       | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && chmod a+r /etc/apt/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
       > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        docker-ce-cli docker-compose-plugin docker-buildx-plugin \
    && rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# 5) GitHub CLI
# ---------------------------------------------------------------------------
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
       | gpg --dearmor -o /etc/apt/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
       > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# 6) Toolchain Go (versiones pinneadas — coinciden con el host)
# ---------------------------------------------------------------------------
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} \
    && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@${MIGRATE_VERSION} \
    && go install github.com/evilmartians/lefthook@${LEFTHOOK_VERSION} \
    && go install golang.org/x/tools/gopls@latest \
    && golangci-lint --version \
    && sqlc version \
    && migrate -version \
    && lefthook version

# ---------------------------------------------------------------------------
# 7) Claude Code CLI + Expo CLI globales
# ---------------------------------------------------------------------------
RUN npm install -g @anthropic-ai/claude-code @expo/cli \
    && claude --version || true

# ---------------------------------------------------------------------------
# 8) Bash prompt + aliases utiles
# ---------------------------------------------------------------------------
RUN cat >> /root/.bashrc <<'EOF'
export PS1="\[\e[1;36m\]ph-dev\[\e[0m\] \[\e[1;33m\]\w\[\e[0m\] $ "
alias ll="ls -lah --color=auto"
alias gs="git status"
alias gd="git diff"
alias k="kubectl"
# Activar pnpm path en shells interactivos
export PATH="/root/.local/share/pnpm:$PATH"
EOF

# ---------------------------------------------------------------------------
# 9) Workdir + entrypoint
# ---------------------------------------------------------------------------
WORKDIR /workspace

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD test -d /workspace || exit 1

# Container always-on: el usuario hace `docker compose exec dev bash`
CMD ["sleep", "infinity"]
