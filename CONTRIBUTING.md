# Contributing to GoTestWAF

Thank you for your interest in contributing to GoTestWAF! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Adding Test Cases](#adding-test-cases)

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all experience levels.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/gotestwaf.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

### Prerequisites

- Go version specified in `go.mod`
- `golangci-lint` and `goimports` (for `make lint` and `make fmt`)
- Docker (optional, for containerized testing)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/wallarm/gotestwaf.git
cd gotestwaf

# Build the binary
make gotestwaf_bin

# Build the Docker image
make gotestwaf

# Run tests
make test

# Run linter and formatter
make lint
make fmt
```

To try your changes against a local ModSecurity instance, see `make modsec` and `make scan_local`.

## How to Contribute

### Reporting Bugs

- Use the GitHub issue tracker
- Include Go version, OS, and steps to reproduce
- Provide expected vs actual behavior

### Suggesting Features

- Open an issue describing the feature
- Explain the use case and potential implementation

### Code Contributions

1. Ensure your code follows Go conventions (`make fmt`, `make lint`)
2. Add tests for new functionality
3. Update documentation as needed

## Pull Request Process

1. Update the README.md if needed
2. Ensure all tests pass
3. Request review from maintainers
4. Address feedback promptly

## Adding Test Cases

Test cases are defined in the `testcases/` directory as YAML files, grouped into test sets:

- `owasp/` — OWASP Top 10 attacks
- `owasp-api/` — OWASP API Security Top 10 attacks
- `community/` — community contributed test cases
- `false-pos/` — legitimate requests that must not be blocked (false positive checks)

```yaml
payload:
  - "your-malicious-payload"
encoder:
  - URL
  - Base64Flat
placeholder:
  - URLParam
  - Header
type: Your Attack Type
```

Payloads must be quoted YAML strings. The full list of supported encoders and placeholders is in the [README](README.md#how-it-works).

### Guidelines for Test Cases

- Use realistic attack payloads
- Cover multiple encoding methods
- Test various HTTP locations (headers, params, body)
- Document the attack type clearly

---

Questions? Open an issue or reach out to the maintainers!
