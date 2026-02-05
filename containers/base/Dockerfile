# Oubliette Base Container - Minimal runtime environment
# Build stage for Go binaries
FROM golang:1.24-bookworm AS builder

WORKDIR /build

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build relay and client binaries (static for portability)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /oubliette-relay ./cmd/oubliette-relay
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /oubliette-client ./cmd/oubliette-client

# Runtime stage - minimal base image
FROM debian:bookworm-slim

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install essential tools only
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean && \
    echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        # Essential tools
        ca-certificates curl wget gnupg sudo \
        # Version control
        git \
        # Shell
        bash locales \
        # Network tools
        openssh-client \
        # JSON tools
        jq \
        # Process management
        procps && \
    # Setup locale
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen && \
    # Cleanup
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=gogol

RUN groupadd -g ${GROUP_ID} ${USERNAME} || true && \
    useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME} && \
    echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} && \
    chmod 0440 /etc/sudoers.d/${USERNAME}

# Switch to user
USER ${USERNAME}
WORKDIR /home/${USERNAME}

# Install Factory AI Droid CLI
RUN curl -fsSL https://app.factory.ai/cli | sh && \
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# Install OpenCode CLI (alternative agent runtime)
RUN curl -fsSL https://opencode.ai/install | bash && \
    echo 'export PATH="$HOME/.opencode/bin:$PATH"' >> ~/.bashrc

# Copy relay and client binaries from builder
COPY --from=builder /oubliette-relay /usr/local/bin/oubliette-relay
COPY --from=builder /oubliette-client /usr/local/bin/oubliette-client

# Copy container init script
COPY scripts/container-init.sh /usr/local/bin/container-init.sh
RUN sudo chmod +x /usr/local/bin/container-init.sh

# Create /mcp directory for relay socket
RUN sudo mkdir -p /mcp && sudo chown ${USERNAME}:${USERNAME} /mcp

ENV PATH="/home/${USERNAME}/.local/bin:/home/${USERNAME}/.factory/bin:/home/${USERNAME}/.opencode/bin:${PATH}"

# Set up workspace directory
RUN mkdir -p /home/${USERNAME}/workspace

# Default working directory
WORKDIR /workspace

SHELL ["/bin/bash", "-c"]

# Use init script as entrypoint to start relay
ENTRYPOINT ["/usr/local/bin/container-init.sh"]
CMD ["sleep", "infinity"]
