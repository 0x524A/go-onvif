# GitHub Actions Workflows

This directory contains all CI/CD workflows for the ONVIF Go library.

## Workflows

### 🔄 CI (`ci.yml`)
Main continuous integration workflow that runs on every push and pull request.

**Jobs:**
- **validate** - Quick validation (formatting, vet, lint)
- **test** - Run tests with coverage on Go 1.23
- **test-matrix** - Test on multiple Go versions (1.21, 1.22, 1.23) and platforms (Linux, macOS, Windows)
- **build** - Build verification for all packages and examples
- **sonarcloud** - Code quality analysis (runs on master/main only)

**Triggers:**
- Push to `master`, `main`, `develop`
- Pull requests to `master`, `main`, `develop`

### 🧪 Extended Tests (`test.yml`)
Extended testing workflow for comprehensive test coverage.

**Jobs:**
- **test-older-versions** - Test on older Go versions (1.19, 1.20)
- **benchmark** - Run benchmark tests
- **race-detector** - Extended race detector tests

**Triggers:**
- Manual dispatch
- Weekly schedule (Sunday 2 AM UTC)
- Push to `master`/`main` when Go files change

### 📊 Coverage Analysis (`coverage.yml`)
Post-CI coverage analysis and reporting.

**Jobs:**
- **coverage-analysis** - Detailed coverage analysis with package breakdown

**Triggers:**
- After successful CI workflow on `master`/`main`

### 🚀 Release (`release.yml`)
Automated release workflow for creating GitHub releases.

**Jobs:**
- **build** - Build binaries for all platforms (Linux, Windows, macOS, multiple architectures)
- **release** - Create GitHub release with artifacts
- **docker** - Build and push Docker images to GHCR

**Triggers:**
- Push tags matching `v*.*.*`
- Manual dispatch with version input

### 🔍 Lint (`lint.yml`)
Dedicated linting workflow.

**Triggers:**
- Push to `master`, `main`, `develop`
- Pull requests

### 🔒 Security (`security.yml`)
Security scanning workflow.

**Jobs:**
- **gosec** - Security scanner
- **govulncheck** - Vulnerability checker

**Triggers:**
- Push to `master`/`main`
- Pull requests
- Weekly schedule

### 📚 Documentation (`docs.yml`)
Documentation validation workflow.

**Triggers:**
- Push to `master`/`main` when docs change
- Manual dispatch

### 🔐 Dependency Review (`dependency-review.yml`)
Dependency vulnerability review.

**Triggers:**
- Pull requests

## Workflow Status

All workflows use:
- ✅ Latest action versions
- ✅ Go 1.23 as primary version
- ✅ Caching for faster builds
- ✅ Matrix builds for multiple platforms
- ✅ Artifact uploads for coverage and releases

## Required Secrets

- `CODECOV_TOKEN` - For coverage reporting (optional)
- `SONAR_TOKEN` - For SonarCloud analysis (optional)
- `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` - For Docker Hub (optional)

## Concurrency

Workflows use concurrency groups to cancel in-progress runs when new commits are pushed, saving CI resources.

---

*Last Updated: December 2, 2025*

