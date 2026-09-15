# GitHub project publication module

## Responsibility

The publication module presents the repository as a conventional, discoverable
bilingual open-source project. The repository README is the primary user entry point;
there is deliberately no separate marketing site or blog.

## Interfaces

- `README.md` and `README.en.md`: primary Chinese and English project pages, including
  direct platform downloads, quick navigation, usage and limitations.
- `docs/assets/preview.svg`: privacy-safe product preview used by both READMEs.
- `.github/workflows/release.yml`: tests, builds, packages, and attaches four archives
  to tagged GitHub Releases.
- `build-all.sh` and `package-release.sh`: local and CI entry points; output contracts
  are `build/` for raw binaries and `release/` for distributable archives.
- Every archive includes Chinese and English instructions plus the MIT license.

## Dependencies and lifecycle

The release workflow depends on GitHub Actions and Go modules declared in `go.mod`.
A maintainer publishes a release by pushing a tag such as `v0.5.0`; GitHub owns the
generated artifacts. Local outputs remain ignored by Git.

## Security boundary

No cookie, user media, account name, or local absolute path may appear in committed
preview assets or workflow artifacts. The website only links to GitHub Releases and
does not proxy downloads or collect analytics.
