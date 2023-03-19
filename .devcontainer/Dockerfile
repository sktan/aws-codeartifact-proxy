FROM mcr.microsoft.com/devcontainers/go:1.18
ENV EDITOR=vim

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
    apt-get install -y --no-install-recommends vim gnupg2 ripgrep && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
