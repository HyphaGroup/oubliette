# Oubliette - Headless Factory AI Droid execution environment

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

# Runtime stage
FROM debian:bookworm

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install system dependencies and development tools
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean && \
    echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        # Essential tools
        ca-certificates curl wget gnupg lsb-release sudo \
        # Development tools
        git vim nano \
        # Build tools
        build-essential gcc g++ make cmake pkg-config \
        # Shell
        bash bash-completion locales \
        # Network tools
        openssh-client \
        # Archive tools
        zip unzip tar gzip bzip2 xz-utils \
        # JSON/YAML tools
        jq yq \
        # Process management
        procps psmisc \
        # Python build dependencies
        python3-dev python3-pip python3-venv \
        libssl-dev libffi-dev \
        # Java dependencies
        default-jdk maven gradle \
        # Search tools
        ripgrep fd-find && \
    # Setup locale
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen && \
    # Cleanup
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
    gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod 644 /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    apt-get install -y gh && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install GitLab CLI
RUN ARCH=$(dpkg --print-architecture) && \
    GLAB_VERSION=$(curl -sL "https://gitlab.com/api/v4/projects/34675721/releases/permalink/latest" | sed -n 's/.*"tag_name":"v\?\([^"]*\)".*/\1/p') && \
    echo "Installing glab version ${GLAB_VERSION} for ${ARCH}" && \
    curl -fsSL -o /tmp/glab.deb \
        "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB_VERSION}/downloads/glab_${GLAB_VERSION}_linux_${ARCH}.deb" && \
    dpkg -i /tmp/glab.deb || apt-get install -f -y && \
    rm /tmp/glab.deb

# Create non-root user
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=gogol

RUN groupadd -g ${GROUP_ID} ${USERNAME} || true && \
    useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME} && \
    echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} && \
    chmod 0440 /etc/sudoers.d/${USERNAME}

# Switch to user for language installations
USER ${USERNAME}
WORKDIR /home/${USERNAME}

# Install uv for Python package management
RUN curl -LsSf https://astral.sh/uv/install.sh | sh && \
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# Install Node.js via NVM
ENV NVM_DIR="/home/${USERNAME}/.nvm"
RUN NVM_VERSION=$(curl -s https://api.github.com/repos/nvm-sh/nvm/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/') && \
    echo "Installing nvm version ${NVM_VERSION}" && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh | bash && \
    . "$NVM_DIR/nvm.sh" && \
    nvm install --lts && \
    nvm alias default node && \
    nvm use default

# Setup NVM in bash
RUN echo 'export NVM_DIR="$HOME/.nvm"' >> ~/.bashrc && \
    echo '[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"' >> ~/.bashrc && \
    echo '[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"' >> ~/.bashrc

# Install OpenSpec CLI globally via npm
RUN . "$NVM_DIR/nvm.sh" && npm install -g @fission-ai/openspec@latest

# Install SDKMAN for Java version management
ENV SDKMAN_DIR="/home/${USERNAME}/.sdkman"
RUN curl -s "https://get.sdkman.io?rcupdate=false" | bash && \
    bash -c "source $SDKMAN_DIR/bin/sdkman-init.sh"

# Setup SDKMAN in bash
RUN echo 'export SDKMAN_DIR="$HOME/.sdkman"' >> ~/.bashrc && \
    echo '[[ -s "$SDKMAN_DIR/bin/sdkman-init.sh" ]] && source "$SDKMAN_DIR/bin/sdkman-init.sh"' >> ~/.bashrc

# Install Factory AI Droid CLI
RUN curl -fsSL https://app.factory.ai/cli | sh && \
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# Copy relay and client binaries from builder
COPY --from=builder /oubliette-relay /usr/local/bin/oubliette-relay
COPY --from=builder /oubliette-client /usr/local/bin/oubliette-client

# Copy container init script
COPY scripts/container-init.sh /usr/local/bin/container-init.sh
RUN sudo chmod +x /usr/local/bin/container-init.sh

# Create /mcp directory for relay socket
RUN sudo mkdir -p /mcp && sudo chown ${USERNAME}:${USERNAME} /mcp

ENV PATH="/home/${USERNAME}/.local/bin:/home/${USERNAME}/.factory/bin:${PATH}"

# Set up workspace directory
RUN mkdir -p /home/${USERNAME}/workspace

# Default working directory
WORKDIR /workspace

# Default to bash (simpler than zsh for headless)
SHELL ["/bin/bash", "-c"]

# Use init script as entrypoint to start relay
ENTRYPOINT ["/usr/local/bin/container-init.sh"]
CMD ["sleep", "infinity"]
