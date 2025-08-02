# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

zgsync is a command-line tool that posts help center content written in Markdown to Zendesk via the Zendesk Help Center REST API. It converts Markdown to HTML for API compatibility and manages both Articles and Translations.

### Tech Stack
- **Language**: Go 1.23.0+ (toolchain go1.24.3)
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

## Essential Commands

### Build & Test
```bash
make build          # Build binary to dist/zgsync
make test           # Run all tests with verbose output
make lint           # Run golangci-lint (must be installed)
make clean          # Clean build artifacts
```

### Run Single Test
```bash
go test -v -run TestName ./internal/package/...
```

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

Expects `~/.config/zgsync/config.yaml` with:
- Required: subdomain, email (with /token suffix), token, default_locale, default_permission_group_id
- Optional: contents_dir, enable_link_target_blank, notify_subscribers

## Main Features

1. **push**: Update Translations or Articles to Zendesk
2. **pull**: Retrieve Translations or Articles from Zendesk
3. **empty**: Create empty draft articles
4. **version**: Show version information

## Project Structure

```
zgsync/
├── cmd/zgsync/         # Main application entry point
├── internal/           # Internal packages
│   ├── cli/           # CLI commands and configuration
│   ├── converter/     # Markdown/HTML conversion logic
│   └── zendesk/       # Zendesk API client and models
├── .github/workflows/ # GitHub Actions CI configuration
├── Makefile          # Build and development commands
└── go.mod            # Go module definition
```