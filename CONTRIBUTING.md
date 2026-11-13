# Contributing to Sayki

Thanks for helping out. Bug reports, fixes and features are all welcome.

## Contributor License Agreement

Sayki is dual-licensed: AGPL-3.0 for everyone, plus a commercial license for
companies that can't meet the AGPL's terms. To keep offering both, every
contributor must sign a Contributor License Agreement (CLA). It gives the
maintainers the right to license your contribution under both licenses. You keep
the copyright to your work.

When you open your first pull request, a bot will ask you to sign the CLA with
one click. Pull requests can't be merged until it's signed.

## Development

```sh
make test   # run the Go tests with the race detector
make run    # build and launch the app
```

- Keep changes focused; one topic per pull request.
- Add or update tests for new logic.
- Never log transcript text or audio.
