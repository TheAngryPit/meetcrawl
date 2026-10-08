# Contributing

## Fixtures

Use only synthetic fixtures. A fixture has no real meeting, no real name, and no personal data.

## Done

Open a pull request. On a clean Linux runner with no network credentials, this command must exit 0:

```sh
make check && scripts/proof.sh
```

Phase 1 ships `scripts/proof.sh` at the repository root.
