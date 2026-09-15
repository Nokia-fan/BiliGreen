# Publication module

## Responsibility

The publication module turns the source repository into a discoverable bilingual
product and downloadable releases without coupling publishing logic to the app.

## Interfaces

- `docs/index.html`: static Chinese/English product page deployed by GitHub Pages.
- `docs/assets/preview.svg`: privacy-safe product preview used by the page and README.
- `.github/workflows/pages.yml`: enables GitHub Pages when necessary, then deploys
  `docs/` after changes reach `main`.
- `.github/workflows/release.yml`: tests, builds, packages, and attaches four archives
  to tagged GitHub Releases.
- `build-all.sh` and `package-release.sh`: local and CI entry points; output contracts
  are `build/` for raw binaries and `release/` for distributable archives.
- Every archive includes Chinese and English instructions plus the MIT license.

## Dependencies and lifecycle

The workflows depend on GitHub Actions and Go modules declared in `go.mod`. Pages is
static and has no runtime backend. A maintainer publishes a release by pushing a tag
such as `v0.5.0`; GitHub owns the generated release artifacts. Local outputs remain
ignored by Git.

## Security boundary

No cookie, user media, account name, or local absolute path may appear in committed
preview assets or workflow artifacts. The website only links to GitHub Releases and
does not proxy downloads or collect analytics.
