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

- Go 1.20 or higher
- Docker (optional, for containerized testing)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/wallarm/gotestwaf.git
cd gotestwaf

# Build the binary
go build -o gotestwaf ./cmd/

# Run tests
go test ./...
```

## How to Contribute

### Reporting Bugs

- Use the GitHub issue tracker
- Include Go version, OS, and steps to reproduce
- Provide expected vs actual behavior

### Suggesting Features

- Open an issue describing the feature
- Explain the use case and potential implementation

### Code Contributions

1. Ensure your code follows Go conventions (`go fmt`, `go vet`)
2. Add tests for new functionality
3. Update documentation as needed

## Pull Request Process

1. Update the README.md if needed
2. Ensure all tests pass
3. Request review from maintainers
4. Address feedback promptly

## Adding Test Cases

Test cases are defined in `testcases/` directory as YAML files:

```yaml
payload:
  - "your-malicious-payload"
encoder:
  - URL
  - Base64Flat
placeholder:
  - UrlParam
  - Header
type: Your Attack Type
```

### Guidelines for Test Cases

- Use realistic attack payloads
- Cover multiple encoding methods
- Test various HTTP locations (headers, params, body)
- Document the attack type clearly

---

Questions? Open an issue or reach out to the maintainers!
