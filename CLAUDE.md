# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

zgsync is a command-line tool that posts help center content written in Markdown to Zendesk via the Zendesk Help Center REST API. It converts Markdown to HTML for API compatibility and manages both Articles and Translations.

### Tech Stack
- **Language**: Go 1.24.0+ (toolchain go1.25.2)
- **Build System**: GNU Make
- **Package Manager**: Go modules
- **CI/CD**: GitHub Actions

### Key Dependencies
- `JohannesKaufmann/html-to-markdown`: HTML to Markdown conversion
- `PuerkitoBio/goquery`: HTML parsing
- `adrg/frontmatter`: Frontmatter parsing for Markdown files
- `alecthomas/kong`: CLI framework
- `yuin/goldmark`: Markdown to HTML conversion (CommonMark compliant)
- `stefanfritsch/goldmark-fences`: Fence blocks support for goldmark

## Environment Setup

### Prerequisites
- Go 1.24.0 or later
- `golangci-lint` (required for `make lint`)

### Install golangci-lint
```bash
# macOS
brew install golangci-lint

# Other platforms: https://golangci-lint.run/welcome/install/
```

### First-time Setup
```bash
git clone https://github.com/tukaelu/zgsync.git
cd zgsync
go mod download
make build
```

## Essential Commands

### Build & Test
```bash
make build          # Build binary to dist/zgsync
make test           # Run all tests with verbose output
make lint           # Run golangci-lint (must be installed)
make clean          # Clean build artifacts
```

### Run Single Test

Replace `TestName` with the target test name and choose the relevant package:

```bash
go test -v -run TestName ./internal/cli/...
go test -v -run TestName ./internal/converter/...
go test -v -run TestName ./internal/zendesk/...
```

## Testing

### Test Patterns
- **Unit tests**: Per-package tests alongside source files
- **Mock-based tests**: Mock HTTP server tests using `internal/zendesk/mock_server.go`
- **Table-driven tests**: Used throughout for parameterized test cases

### Run Tests by Package
```bash
go test -v ./internal/cli/...
go test -v ./internal/converter/...
go test -v ./internal/zendesk/...
```

### Test Utilities
`internal/testutil/` provides shared helpers used across packages:
- `assertions.go`: Custom assertion helpers
- `comparator.go`: Value comparison utilities
- `filehelper.go`: File-based test helpers
- `httphelper.go`: HTTP mock/helper utilities

## Architecture

### Core Components

1. **CLI Layer** (`internal/cli/`)
   - Uses Kong framework for command parsing
   - Commands embed `Global` struct for shared config
   - Each command (push/pull/empty) has its own struct and Run method

2. **Zendesk Client** (`internal/zendesk/`)
   - `Article` and `Translation` types handle API data models
   - Methods: `FromFile()`, `FromJson()`, `Save()`, `ToPayload()`
   - Files saved as: `{ArticleID}-{Locale}.md` (translations) or `{ArticleID}.md` (articles)

3. **Converter** (`internal/converter/`)
   - Markdown→HTML: Uses goldmark (CommonMark compliant)
   - HTML→Markdown: Uses html-to-markdown
   - Supports Pandoc-style divs: `:::{.class}` and heading attributes: `## Title {#id}`

### Data Flow

1. **Pull**: Zendesk API → JSON → Article/Translation struct → Markdown file with frontmatter
2. **Push**: Markdown file → Parse frontmatter → Convert body to HTML → API payload → Zendesk

### File Format

Translation files contain:
```markdown
---
title: Title
locale: ja
draft: true
section_id: 123
---
## Markdown content
```

Article files contain only frontmatter metadata (body ignored).

## Configuration

Expects `~/.config/zgsync/config.yaml` with the fields listed below.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subdomain` | string | Yes | Zendesk subdomain |
| `email` | string | Yes | Zendesk email (`user@example.com/token` format) |
| `token` | string | Yes | Zendesk API token |
| `default_locale` | string | Yes | Default locale for articles |
| `default_permission_group_id` | int | Yes | Default permission group ID |
| `contents_dir` | string | No | Path to contents directory (default: `.`) |
| `enable_link_target_blank` | bool | No | Open links in new tab (default: false) |
| `notify_subscribers` | bool | No | Notify subscribers on create/update (default: false) |
| `default_comments_disabled` | bool | No | Disable comments by default (default: false) |
| `default_user_segment_id` | *int | No | Default user segment ID (nil means unset) |

## Main Features

1. **push**: Update Translations or Articles to Zendesk
2. **pull**: Retrieve Translations or Articles from Zendesk
3. **empty**: Create empty draft articles
4. **version**: Show version information

## CI/CD & Release Flow

Releases are automated via GitHub Actions (`.github/workflows/tagpr.yml`):

1. Push to `main` triggers [tagpr](https://github.com/Songmu/tagpr), which auto-generates a release tag
2. When a tag is created, [GoReleaser](https://goreleaser.com/) builds binaries for multiple platforms
3. Release artifacts are published to GitHub Releases and distributed via Homebrew tap

## Project Structure

```
zgsync/
├── cmd/zgsync/         # Main application entry point
├── internal/           # Internal packages
│   ├── cli/           # CLI commands and configuration
│   ├── converter/     # Markdown/HTML conversion logic
│   ├── zendesk/       # Zendesk API client and models
│   └── testutil/      # Shared test utilities (assertions, comparator, filehelper, httphelper)
├── .github/workflows/ # GitHub Actions CI configuration
├── .goreleaser.yaml   # GoReleaser configuration for multi-platform builds
├── Makefile          # Build and development commands
└── go.mod            # Go module definition
```

## Claude Code Skills

`.claude/skills/` contains skills for common development tasks:
- `go-test`: Run tests and analyze failures
- `go-lint-fix`: Code quality checks and auto-formatting
- `add-cli-command`: Scaffold a new Kong-based CLI command
- `git-commit`: Create Conventional Commits format commits
- `git-pr`: Create or update pull requests
