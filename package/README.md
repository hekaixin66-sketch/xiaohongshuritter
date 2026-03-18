# Package Templates

This directory contains deployment wrappers for `xiaohongshuritter`.

## openclaw-lite

Use `package/openclaw-lite` when you want a standard Docker-based deployment wrapper for Linux amd64 or general server environments. It builds from the current repository using the top-level `Dockerfile`.

## openclaw-source-lite

Use `package/openclaw-source-lite` when you want an OpenClaw-friendly ARM64 package that builds from the current repository using `Dockerfile.arm64`. This package is the recommended choice when:
- you deploy on Apple Silicon or ARM64 hosts
- Chromium runtime download is blocked
- you need a more stable OpenClaw handoff package

## Usage Model

Both folders are designed to live inside this repository. After cloning the repository, enter the package directory and run its scripts.
