# Oubliette OSINT Container - Reconnaissance and intelligence tools
# Extends base with OSINT tooling
FROM oubliette-base:latest

USER root

# Install Python and Go (for Go-based OSINT tools)
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        python3 python3-pip python3-venv python3-dev \
        golang-go \
        whois dnsutils nmap netcat-openbsd \
        libxml2-dev libxslt1-dev zlib1g-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Switch to user for tool installations
USER gogol
WORKDIR /home/gogol

# Set up Go environment
ENV GOPATH="/home/gogol/go"
ENV PATH="${GOPATH}/bin:${PATH}"
RUN mkdir -p ${GOPATH}/bin && \
    echo 'export GOPATH="$HOME/go"' >> ~/.bashrc && \
    echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.bashrc

# Install Python OSINT libraries
RUN pip3 install --user --break-system-packages \
    requests beautifulsoup4 lxml \
    dnspython \
    shodan \
    theHarvester \
    censys \
    ipwhois \
    whois

# Install Go-based OSINT tools
RUN go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest && \
    go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest && \
    go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest && \
    go install -v github.com/owasp-amass/amass/v4/...@master

# Add Python user bin to PATH
RUN echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

WORKDIR /workspace
