## Git Hooks

Install the repo-managed hooks with:

```sh
./scripts/install-hooks.sh
```

The pre-commit hook will:

- run `gofmt -w` on staged Go files and restage them
- run `go vet ./...`
