# Zendesk API Integration Details

## Overview
zgsync integrates with Zendesk Help Center REST API to manage articles and translations.

## Configuration
Configuration file location: `~/.config/zgsync/config.yaml`

Required fields:
- `subdomain`: Zendesk subdomain
- `email`: Email address with "/token" suffix
- `token`: API token
- `default_comments_disabled`: Default comment setting
- `default_locale`: Default locale (e.g., "ja")
- `default_permission_group_id`: Permission group ID

Optional fields:
- `default_user_segment_id`: User segment ID
- `notify_subscribers`: Notify subscribers on update
- `contents_dir`: Local directory for articles
- `enable_link_target_blank`: Open links in new tab

## File Formats

### Translation Files
Format: `{Article ID}-{Locale}.md`
Contains:
- Frontmatter with metadata (title, locale, draft, etc.)
- Markdown body content

### Article Files  
Format: `{Article ID}.md`
Contains:
- Frontmatter with full article metadata
- No body content (ignored if present)

## Markdown/HTML Conversion
- Uses goldmark for Markdown to HTML (CommonMark compliant)
- Supports Pandoc-style div notation: `:::{.class} content :::`
- Supports attribute specification: `## Title {#id .class}`
- Uses html-to-markdown for reverse conversion

## API Operations
- Create/Update Translations
- Create/Update Articles
- Pull existing content
- Create empty draft articles