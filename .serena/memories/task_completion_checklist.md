# Task Completion Checklist for zgsync

When completing a coding task in the zgsync project, ensure you:

## 1. Code Quality Checks
- [ ] Run `make test` to ensure all tests pass
- [ ] Run `make lint` to check for linting issues
- [ ] Fix any linting warnings or errors

## 2. Build Verification
- [ ] Run `make build` to ensure the project builds successfully
- [ ] Test the built binary if relevant to your changes

## 3. Code Review
- [ ] Ensure code follows Go conventions and project style
- [ ] Check that error handling is proper (no ignored errors)
- [ ] Verify no sensitive information is logged or exposed
- [ ] Ensure proper struct tags for YAML/JSON marshaling

## 4. Documentation
- [ ] Update code comments if needed
- [ ] Update README.md if adding new features or changing behavior

## 5. Before Committing
- [ ] Run full test suite: `make test`
- [ ] Run linter: `make lint`
- [ ] Ensure clean build: `make build`

## Important Commands Summary
```bash
make test   # Run tests
make lint   # Run linter
make build  # Build binary
```

If any of these steps fail, fix the issues before considering the task complete.