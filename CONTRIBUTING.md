# Contributing to go-remove

Thank you for your interest in contributing to **go-remove**!
This document is intended to provide comprehensive guidelines for contributing to this project.

## Table of Contents

1. [Introduction](#introduction)
2. [Development Environment Setup](#development-environment-setup)
3. [Commit Signing Requirement](#commit-signing-requirement)
4. [AI Policy](#ai-policy)
5. [Code Standards](#code-standards)
6. [Testing Requirements](#testing-requirements)
7. [Linting and Validation](#linting-and-validation)
8. [Pull Request Process](#pull-request-process)
9. [Development Workflow](#development-workflow)
10. [Commit Message Guidelines](#commit-message-guidelines)
11. [Mock Generation](#mock-generation)
12. [Release Process](#release-process)
13. [Questions and Support](#questions-and-support)

---

## Introduction

**go-remove** is a CLI tool designed to safely remove Go binaries with undo and history support.
Contributions from the community are welcomed and your efforts to improve this project are appreciated!

### Code of Conduct

All contributors are expected to:

- Be respectful and professional in all interactions
- Focus on what is best for the project and the OSS community

---

## Development Environment Setup

### Prerequisites

Before you begin, ensure you have the following tools installed:

| Tool          | Version         | Purpose                     |
|---------------|-----------------|-----------------------------|
| Go            | 1.26.0 or later | Core language runtime       |
| Git           | Latest          | Version control             |
| GPG           | Latest          | Commit signing (required)   |
| golangci-lint | Latest          | Linting and static analysis |
| mockery       | Latest          | Mock generation             |
| make          | Latest          | Build automation            |

### Setup Steps

1. **Fork the repository**
   - Visit <https://github.com/nicholas-fedor/go-remove>
   - Click the `Fork` button

2. **Clone your fork**

   ```bash
   git clone https://github.com/<YOUR_USERNAME>/go-remove.git
   cd go-remove
   ```

3. **Configure upstream remote**

   ```bash
   git remote add upstream https://github.com/nicholas-fedor/go-remove.git
   ```

4. **Install dependencies**

   ```bash
   make deps
   ```

5. **Verify your setup**

   ```bash
   make verify
   ```

---

## Commit Signing Requirement

**ALL commits MUST be GPG signed.** Unsigned commits will be automatically rejected by CI.

### Generating a GPG Key

If you don't have a GPG key, generate one:

```bash
gpg --full-generate-key
```

Select these options:

- Key type: RSA and RSA
- Key size: 4096 bits
- Expiration: Your preference (recommended: 2 years)
- Name: Your name
- Email: The email used in your GitHub account

### Configuring Git for Commit Signing

1. **Get your GPG key ID:**

   ```bash
   gpg --list-secret-keys --keyid-format=long
   ```

   Look for the line starting with `sec` (e.g., `sec   rsa4096/ABC123DEF456GHI7`)

2. **Configure Git to sign all commits:**

   ```bash
   git config --global commit.gpgsign true
   git config --global user.signingkey YOUR_KEY_ID
   ```

3. **Configure GPG program (macOS/Linux):**

   ```bash
   git config --global gpg.program gpg
   ```

### Verifying Signed Commits

Verify your commits are signed:

```bash
git log --show-signature -1
```

---

## AI Policy

The use of LLMs and AI-assisted development tools is **permitted** under the following conditions:

### Requirements for AI-Assisted Code

- **Thorough vetting is MANDATORY**: You must review and understand every line of AI-generated code
- **Full understanding required**: You must be able to explain what the code does and why
- **Project standards compliance**: All AI-generated code must follow this project's standards and best practices
- **Contributor responsibility**: You are fully responsible for any AI-generated code you submit
- **Pass all checks**: AI-generated code must pass all linting, testing, and review requirements

### CodeRabbit Review

CodeRabbit is enabled for all PRs and will review all submissions, including AI-assisted code.
You must address all issues identified by CodeRabbit.

---

## Code Standards

### Go Version and Features

This project targets **Go 1.26+** and leverages modern language features:

- **Generics**: Use for type-safe generic programming
- **Iterators**: Use for memory-efficient iteration
- **Range-over-func**: Apply for custom iteration patterns
- **Error wrapping**: Use improved error handling with `fmt.Errorf` and `%w`

### Style Guide

Follow **Uber's Go Style Guide** (<https://github.com/uber-go/guide>).

### Naming Conventions

| Type                   | Convention             | Example            |
|------------------------|------------------------|--------------------|
| Exported identifiers   | MixedCaps              | `MyFunction`       |
| Unexported identifiers | mixedCaps              | `myFunction`       |
| Package names          | Lowercase, single word | `storage`          |
| Interface names        | End with 'er'          | `Reader`, `Writer` |

### Error Handling

- Use `errors.Is()` for error checking (not `==`)
- Wrap errors with `fmt.Errorf` using `%w` verb
- Return errors early; don't bury them in conditionals
- Pass `context.Context` as the first parameter to functions

### Documentation

- All exported identifiers must be documented
- Larger packages require a `doc.go` file
- Comments should be clear and concise
- Place inline comments above the code (not to the side)

### Package Organization

- Public packages: Place under `pkg/` directory
- Internal packages: Place under `internal/` directory

---

## Testing Requirements

Comprehensive testing is **MANDATORY**. All tests must pass before submission.

### Test Types

#### Unit Tests (White-Box)

- **Location**: Same package as code, in `*_test.go` files
- **Requirements**:
  - No external calls allowed (use interfaces and mocks)
  - Use table-driven tests for multiple cases
  - Test error conditions thoroughly
- **Coverage Target**: 80%+ for new code

#### Integration Tests (Black-Box)

- **Location**: `/testing/integration/<package>/`
- **Requirements**:
  - Test from outside the package
  - Use Mockery-generated mocks
  - No external calls allowed
  - Package name: <package>_test
- **Purpose**: Verify component interactions

#### E2E Tests

- **Location**: `/testing/e2e/<package>/`
- **Requirements**:
  - Test against actual external systems
  - Validate real-world behavior
  - Package name: `e2e`
- **Purpose**: Full system validation

### Coverage Requirements

| Scope        | Minimum Coverage |
|--------------|------------------|
| Project      | 50%              |
| PR Changes   | 70%              |
| New Features | 80%              |

**Note**: Mock files are excluded from coverage calculations.

### Test Commands

```bash
# Run all tests with race detection
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e

# Generate coverage report
make coverage

# Run benchmarks
make benchmark
```

---

## Linting and Validation

Code must pass all linting checks before submission.

### Pre-Submission Checklist

Run these commands before submitting a PR:

1. **Lint check** (MUST pass without `--fix` flag):

   ```bash
   make lint
   ```

2. **Vet check** (additional static analysis):

   ```bash
   make vet
   ```

3. **Run all tests** (with race detection):

   ```bash
   make test
   ```

4. **Verify coverage**:

   ```bash
   make coverage
   ```

### Important Notes

- Any use of the `nolint` directive should include a corresponding comment.
- Updates to the Golangci-lint configuration file (`.golangci.yaml`) are permitted on an as-needed basis to address issues, such as reducing false positives or increasing compliance with modern Go best practices.

---

## Pull Request Process

### Requirements

All PRs must meet the following criteria:

1. **Target branch**: Submit against `main`
2. **PR title**: Use Conventional Commit syntax (e.g., `feat:`, `fix:`, `docs:`)
3. **Signed commits**: All commits must be GPG signed
4. **CI checks**: All checks must pass
5. **CodeRabbit**: All CodeRabbit issues must be addressed
6. **Coverage**: Minimum 70% coverage on changed code

### PR Body Format

The following format should be used for most PR's related to feature additions or bug reports:

```markdown
Brief 2-3 sentence description of the change

## Problem

Description of the problem being solved

## Solution

Description of the solution implemented

## Changes

- Bullet point list of changes (max 5 items)
- Organized by significance (most important first)
```

### CodeRabbit Compliance

- CodeRabbit is enabled for **all PRs**
- CodeRabbit identifies issues with:
  - Code quality
  - Security vulnerabilities
  - Best practices violations
- You **MUST** work through **ALL** issues identified by CodeRabbit
- Do not dismiss CodeRabbit comments without addressing them

---

## Development Workflow

### Step-by-Step Workflow

1. **Create a feature branch:**

   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes:**
   - Follow Go 1.26+ conventions
   - Write self-documenting code

3. **Write/update tests:**
   - Unit tests (required)
   - Integration tests (as appropriate)
   - E2E tests (for feature changes)

4. **Generate mocks** (if interfaces changed):

   ```bash
   make mocks
   ```

5. **Run validation:**

   ```bash
   make check
   make test
   ```

6. **Commit with signing:**

   ```bash
   git commit -S -m "feat(scope): your message"
   ```

7. **Push to your fork:**

   ```bash
   git push origin feat/your-feature-name
   ```

8. **Open a PR:**
   - Use proper title format
   - Fill out the PR body template
   - Ensure all checks pass

---

## Commit Message Guidelines

This follows the Conventional Commit guidelines and Angular's standards.

### Format

```text
<type>(<scope>): <subject>

<body>
```

### Types

| Type       | Description                                      |
|------------|--------------------------------------------------|
| `feat`     | New feature                                      |
| `fix`      | Bug fix                                          |
| `docs`     | Documentation changes                            |
| `style`    | Code style changes (formatting, no logic change) |
| `refactor` | Code refactoring                                 |
| `test`     | Test additions/modifications                     |
| `chore`    | Maintenance tasks                                |
| `perf`     | Performance improvements                         |
| `ci`       | CI/CD changes                                    |
| `build`    | Build system changes                             |

### Guidelines

- Use **imperative tense** ("Add feature" not "Added feature")
- Follow Angular's guidelines for type and scope
- Keep the subject line under 72 characters
- Use the body for detailed explanation
- **Maximum 5 bullet points** in the body
- Group logically and order by significance

### Example

```text
feat(history): add undo functionality for deleted binaries

Implement undo capability for the remove command, allowing users to
restore accidentally deleted Go binaries from the trash.

- Add Undo() method to history.Manager interface
- Implement undo logic in CLI commands
- Add comprehensive tests for undo operations
- Update documentation with usage examples
```

---

## Mock Generation

When you modify interfaces, you must regenerate mocks.

### Configuration

- Configuration file: `.mockery.yaml`
- Mocks stored in: `mocks/` subdirectories within packages

### Regenerating Mocks

```bash
make mocks
```

### Important Notes

- **Never** manually edit generated mocks
- Always regenerate mocks after interface changes
- Ensure mocks are committed with your changes

---

## Release Process

Releases are handled by project maintainers using GoReleaser.

### Supported Platforms

- **Linux**: amd64, i386, armhf, arm64v8
- **Windows**: amd64, i386, arm64v8

### Artifacts

All release artifacts are:

- Signed with GPG
- Checksummed for integrity verification

### Important

- **Do not** modify `.goreleaser.yaml` without maintainer approval

---

## Questions and Support

### Getting Help

- **Bugs**: Open an issue with reproduction steps
- **Feature requests**: Open an issue with detailed description
- **General questions**: Use GitHub Discussions
- **Sensitive matters**: Email <nick@nickfedor.com>

### Before Asking

1. Check existing issues and discussions
2. Review this documentation thoroughly
3. Search closed issues for similar problems

---

## License

By contributing to go-remove, you agree that your contributions will be licensed under the **GNU Affero General Public License v3 (AGPL-3.0)**.

---

Thank you for contributing to go-remove!

---

*Author: Nicholas Fedor <nick@nickfedor.com>*
*Module: github.com/nicholas-fedor/go-remove*
*License: AGPL-3.0*
