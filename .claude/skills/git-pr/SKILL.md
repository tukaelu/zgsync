---
name: git-pr
description: |
  Pushes local changes to remote, analyzes the diff, and auto-generates a PR title and
  description to create or update a pull request. Triggers on requests like "PRを作成",
  "プルリクエストを出して", "PRを更新", "create a PR", "open a pull request", "update PR".
allowed-tools:
  - Bash(git:*)
  - Bash(gh:*)
---

# Pull Request Management

Push local changes and create or update a pull request.

## Requirements

- **`gh` CLI must be installed** (https://cli.github.com)
- **`gh` must be authenticated with GitHub**; run `gh auth login` to set this up

## Rules

- Do not run on the `main` branch (show a warning and abort)
- Write PR title and description in English
- Write PR title concisely in Conventional Commits format (`<type>(<scope>): <subject>`)
- Abort with a warning if there are uncommitted changes

## Workflow

### Step 1. Pre-flight checks

- Run `gh --version` to verify `gh` is installed; if not, guide the user to the installation instructions (https://cli.github.com) and abort
- Run `gh auth status` to verify authentication; if not authenticated, guide the user to run `gh auth login` and abort
- Run `git branch --show-current` to confirm the current branch
- If on `main`, show a warning and abort
- Run `git status` to check for uncommitted changes
- If uncommitted changes exist, warn the user, guide them to commit first, and abort

### Step 2. Review changes

- Run `git log main..HEAD --oneline` to list commits
- Run `git diff main...HEAD --stat` to get change statistics
- Run `git diff main...HEAD` to review the diff

### Step 3. Push to remote

- Push to remote with `git push -u origin HEAD`
- If the push fails, investigate the error and inform the user of the cause (rejected push, permission error, etc.) before aborting

### Step 4. Check for existing PR

- Run `gh pr view --json number,title,body,url` to check for an existing PR
- If a PR exists → proceed to Step 5a (update)
- If no PR exists → proceed to Step 5b (create)

### Step 5a. Update existing PR

- Regenerate the PR title and description from the current diff
- Compare with the existing PR title and description; determine that an update is needed if they no longer match the current changes
- If an update is needed, present the new content to the user and ask for confirmation
- If the user requests changes, regenerate based on their feedback and ask for confirmation again
- Once approved, update with `gh pr edit --title "{title}" --body "{body}"`
- If no update is needed, inform the user and proceed to Step 6

### Step 5b. Create new PR

- Generate a PR title and description from the diff
- Present the content to the user and ask for confirmation
- If the user requests changes, regenerate based on their feedback and ask for confirmation again
- Once approved, create with `gh pr create --base main --title "{title}" --body "{body}"`

### Step 6. Report completion

- Display the PR URL

## PR Description Format

```markdown
## Summary

- {brief bullet points of the changes}

## Changes

{detailed description of the changes}
```

## PR Title and Description Examples

**New feature:**
```
title: feat(cli): add --section-id option to pull command
```
```markdown
## Summary

- Add `--section-id` flag to the `pull` command to filter articles by section

## Changes

Previously, the `pull` command fetched all articles regardless of section.
This change allows users to specify a section ID to retrieve only articles
belonging to that section.
```

**Bug fix:**
```
title: fix(zendesk): fix error when locale is not set in Translation fetch
```
```markdown
## Summary

- Fix a panic that occurred when fetching a Translation with an empty locale field

## Changes

The `FromJson` method did not handle the case where the `locale` field was
missing in the API response. Added a fallback to the default locale from config.
```
