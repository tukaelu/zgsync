# zgsync

Currently under development, and there may be breaking changes.

## Description

zgsync is a command-line tool that posts help center content written in Markdown via the [Zendesk Help Center REST API](https://developer.zendesk.com/api-reference/help_center/help-center-api/introduction/).  
When posting, it converts the Markdown to HTML to match the API interface.

## Installation

### Homebrew tap

```
brew install tukaelu/tap/zgsync
```

### Binary install

Please download the appropriate Zip archive for your environment from the [releases](https://github.com/tukaelu/zgsync/releases).

## Configuration

By default, it references the configuration file at `~/.config/zgsync/config.yaml`, so please create it in advance.
You can also explicitly specify the path using the `--config` option.

```yaml:~/.config/zgsync/config.yaml
subdomain: <your zendesk subdomain>
email: <your zendesk email address>/token
token: <your zendesk token>
default_comments_disabled: true
default_locale: ja
default_permission_group_id: 123
default_user_segment_id: 456
notify_subscribers: false
contents_dir: path/to/contents
```

| Key                         | Required | Description                                              |
| --------------------------- | -------- | -------------------------------------------------------- |
| subdomain                   | true     | Specify a brand-specific subdomain                       |
| email                       | true     | Specify the email address with "/token" added to the end |
| token                       | true     | Specify your API token                                   |
| default_comments_disabled   | true     | Specify the default comments disabled                    |
| default_locale              | true     | Specify the default locale for translations              |
| default_permission_group_id | true     | Specify the default permission group ID                  |
| default_user_segment_id     | true     | Specify the default user segment ID                      |
| notify_subscribers          | false    | Specify whether to notify subscribers of the article     |
| contents_dir                | false    | Specify the local directory path to manage articles      |

## Usage

zgsync consists of the subcommands pull, push, and empty.  
By default, it handles Translations among the data models of the Zendesk Help Center, but it can also handle Articles by specifying a specific option.

zgsync saves Translations in files named `{Article ID}-{Locale}.md`. When using the pull or empty commands, specifying the `--save-article` option saves Articles in files named `{Article ID}.md`.  
When pushing, it does not automatically determine whether it is a Translation or an Article. Therefore, to post an Article, explicitly specify the `--article` option and provide the Article file.

### push

The push subcommand updates posts, either Translations or Articles, to the remote.

```
Usage: zgsync push <files> ... [flags]

Push translations or articles to the remote.

Arguments:
  <files> ...    Specify the files to push.

Flags:
      --article                                  Specify when posting an article. If not specified, the translation will be pushed.
      --dry-run                                  dry run
      --raw                                      It pushes raw data without converting it from Markdown to HTML.
```

### pull

The pull subcommand retrieves translations or articles from the remote and saves them locally.

```
Usage: zgsync pull <article-i-ds> ... [flags]

Pull translations or articles from the remote.

Arguments:
  <article-i-ds> ...    Specify the article IDs to pull.

Flags:
  -l, --locale=STRING                            Specify the locale to pull. If not specified, the default locale will be used.
      --raw                                      It pulls raw data without converting it from HTML to Markdown.
  -a, --save-article                             It pulls and saves the article in addition to the translation.
      --without-section-dir                      It doesn't save in a directory named after the section ID.
```

By default, the pull subcommand saves under `{contents_dir}/{section_id}`. You can also specify an option to output directly under `{contents_dir}/`.
If a Translation or Article already exists at the specified local path, it will be overwritten.

### empty

The empty subcommand creates an empty draft article remotely and saves it locally.

```
Usage: zgsync empty --section-id=INT --title=STRING [flags]

Creates an empty draft article remotely and saves it locally.

Flags:
  -s, --section-id=INT                           Specify the section ID of the article.
  -t, --title=STRING                             Specify the title of the article.
  -l, --locale=STRING                            Specify the locale to pull. If not specified, the default locale will be used.
  -p, --permission-group-id=INT                  Specify the permission group ID. If not specified, the default value will be used.
  -u, --user-segment-id=INT                      Specify the user segment ID. If not specified, the default value will be used.
      --save-article                             It saves the article in addition to the translation.
      --without-section-dir                      It doesn't save in a directory named after the section ID.
```

The empty subcommand should not be used when adding a new Translation to an existing Article.

## Markdown file format

zgsync manages Translations and Articles in the following formats respectively.

### Translations

Translations are files composed of Frontmatter and Markdown text. The Markdown, which corresponds to the body of the article, is written in this file.  
Ensure that the Markdown Frontmatter related to properties required by the API is not missing.
The section_id is included for administrative purposes but is not required by the Translation API.

```markdown
---
title: cool title
locale: ja
draft: true
outdated: false
section_id: 1234567890
source_id: 12345678901234
html_url: https://{your help center domain}/hc/ja/articles/12345678901234
created_at: "2024-01-01T00:00:00Z"
updated_at: "2024-01-01T00:00:00Z"
---
## Markdown

some cool text
```

refs: [Translations | Zendesk Developer Docs](https://developer.zendesk.com/api-reference/help_center/help-center-api/translations/)

### Article

Articles manage only the metadata related to the post in the Frontmatter. Please note that any body text written in this file will be ignored.

```markdown
---
author_id: 98765432109876
comments_disabled: true
content_tag_ids: []
created_at: "2024-01-01T00:00:00Z"
draft: false
edited_at: "2024-01-01T00:00:00Z"
html_url: https://{your help center domain}/hc/ja/articles/12345678901234
id: 12345678901234
label_names: []
locale: ja
outdated: false
outdated_locales: []
permission_group_id: 1234567
position: 0
promoted: false
section_id: 567890123456
source_locale: ja
title: cool title
updated_at: "2024-01-01T00:00:00Z"
url: https://{subdomain}.zendesk.com/api/v2/help_center/ja/articles/12345678901234.json
user_segment_id: 234567890123
user_segment_ids: []
vote_count: 0
vote_sum: 0
---
```

refs: [Articles | Zendesk Developer Docs](https://developer.zendesk.com/api-reference/help_center/help-center-api/articles/)

## Regarding Markdown and HTML conversion

- The conversion from Markdown to HTML is performed by [yuin/goldmark](https://github.com/yuin/goldmark), which adheres to the CommonMark specification.
- It supports the output of div tags using notation similar to Pandoc.

```markdown
:::
messages
:::
```

- For headings such as h1 and h2 tags, as well as div tags, it also supports the specification of attributes.

```markdown
## Hoge {#hoge .h2}   // ==> <h2 id="hoge" class="h2">Hoge</h2>

:::{.block .warning}  // ==> <div class="block warning"><p>warning messages</p></div>
warning messages
:::
```

- The conversion from HTML to Markdown uses [JohannesKaufmann/html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown), so fully consistent bidirectional conversion is not currently supported.

## Contributing

Contributions are very welcome! Feel free to submit issues and pull requests.

## License

MIT License

Copyright (c) 2024 Tsukasa NISHIYAMA
