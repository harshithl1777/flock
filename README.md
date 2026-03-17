## Git Hooks

This repo uses `lefthook` for local Git hooks with named pass/fail output.

Install `lefthook`, then enable the repo-managed hooks with:

```sh
./scripts/install-hooks.sh
```

The pre-commit hook will:

- run `gofmt -w` on staged Go files and restage them
- run `go vet ./...`

If `lefthook` is not installed yet, one option is:

```sh
brew install lefthook
```
