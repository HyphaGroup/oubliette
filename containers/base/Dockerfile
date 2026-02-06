# Oubliette Base Container - Minimal agent runtime environment

# Build stage for Go binaries
FROM golang:1.24-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /oubliette-relay ./cmd/oubliette-relay
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /oubliette-client ./cmd/oubliette-client

# Runtime stage
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# System packages (minimal set for agent runtime)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        locales \
        openssh-client \
        procps && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Ripgrep (both agents use it for code search)
RUN case $(dpkg --print-architecture) in \
        amd64) RG_ARCH="x86_64-unknown-linux-musl" ;; \
        arm64) RG_ARCH="aarch64-unknown-linux-gnu" ;; \
    esac && \
    curl -fsSL "https://github.com/BurntSushi/ripgrep/releases/download/14.1.1/ripgrep-14.1.1-${RG_ARCH}.tar.gz" \
        | tar xz --strip-components=1 -C /usr/local/bin "ripgrep-14.1.1-${RG_ARCH}/rg"

# Copy relay and client binaries
COPY --from=builder /oubliette-relay /usr/local/bin/oubliette-relay
COPY --from=builder /oubliette-client /usr/local/bin/oubliette-client

# Container init script
COPY --chmod=755 scripts/container-init.sh /usr/local/bin/container-init.sh

# Create non-root user
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=gogol

RUN groupadd -g ${GROUP_ID} ${USERNAME} || true && \
    useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME}

# Directories that need user ownership (created as root)
RUN mkdir -p /mcp && chown ${USERNAME}:${USERNAME} /mcp
RUN mkdir -p /workspace && chown ${USERNAME}:${USERNAME} /workspace

# Switch to non-root user
USER ${USERNAME}
WORKDIR /home/${USERNAME}

# Install OpenCode CLI
RUN curl -fsSL https://opencode.ai/install | bash

ENV PATH="/home/${USERNAME}/.local/bin:/home/${USERNAME}/.opencode/bin:${PATH}"

# Verify agent and tools
RUN opencode --version && rg --version

WORKDIR /workspace
SHELL ["/bin/bash", "-c"]
ENTRYPOINT ["/usr/local/bin/container-init.sh"]
CMD ["sleep", "infinity"]
