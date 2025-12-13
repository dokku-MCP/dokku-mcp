FROM mcr.microsoft.com/devcontainers/go:1-1.23-bookworm

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    make \
    git \
    curl \
    wget \
    ssh \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

RUN go install github.com/cosmtrek/air@latest \
    && go install github.com/onsi/ginkgo/v2/ginkgo@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest

ENV PATH="/go/bin:${PATH}"
