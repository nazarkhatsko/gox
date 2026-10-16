# gox

Small, reusable Go packages that extend the standard library.

```sh
go get github.com/nazarkhatsko/gox
```

## Packages

| Package | Description |
| --- | --- |
| [`dbx`](dbx) | Less repetitive `database/sql`: one querier interface, transactions as closures, generic `Get`/`Select`, column type conversions. |
| [`errs`](errs) | Error kinds: classify failures with `errors.Is`, map kinds to transport codes. |
| [`httpx`](httpx) | HTTP client for calling APIs: shared connection pool, default timeout, JSON bodies, whole-body responses. |
| [`tagx`](tagx) | Binds a map of options to a struct via `key`, `default`, `description` and `validate` tags. |

Packages depend only on the standard library, except `tagx` (go-playground/validator). `dbx` tests use modernc.org/sqlite.

## Development

Tasks run with [Task](https://taskfile.dev): `task --list` shows them all.

```sh
task hooks    # once per clone: enable the git hooks below
task check    # format check, lint and tests
task release  # check, tag and push a version
```

Checks run at three points:

| When | What |
| --- | --- |
| `git commit` | format (files are fixed, then the commit stops for review) |
| `git push`, `task release` | format check, lint, tests; the working tree must be clean |
| CI, on every push and pull request | format check, lint, tests; a `v*` tag also gets a GitHub release |
