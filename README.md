# zgsync

This is an alpha version currently under development, and there may be breaking changes.

## Description

zgsync is a command-line tool that posts article translations from Markdown files to Zendesk Guide via the [Zendesk Help Center REST API](https://developer.zendesk.com/api-reference/help_center/help-center-api/introduction/).

## Installation

TODO

## Configuration

```yaml
subdomain: <your zendesk subdomain>
email: <your zendesk email address>/token
token: <your zendesk token>
default_locale: ja
default_permission_group_id: 123
default_user_segment_id: 456
notify_subscribers: false
contents_dir: path/to/contents
```

## License

MIT License

Copyright (c) 2024 Tsukasa NISHIYAMA
