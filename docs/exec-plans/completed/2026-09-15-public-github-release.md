# 2026-09-15 Public GitHub Release

Last Reviewed: 2026-09-15
Status: Completed locally; remote publication blocked
Owner: Project maintainer
Current Slice: Awaiting GitHub authorization
Blocked By: No GitHub CLI authentication and no Git commit identity
Risk Level: Low

## Goal

Prepare BiliGreen as a public, discoverable, bilingual GitHub project whose normal
repository README explains the product clearly, with repeatable release automation.

## Repository findings and decision

The project previously lived as an untracked folder inside a broader workspace Git
tree. It had local build and packaging scripts but no independent history, license,
public security policy, English documentation, product page, or CI workflows.

Two approaches were considered: a manually maintained release repository, or a
source-first repository with GitHub Actions and Pages. The source-first approach was
selected because it keeps binaries reproducible, documentation versioned, and future
releases easier to audit.

## Implemented slices

1. Added repository hygiene, MIT licensing, security and contribution documents.
2. Added Chinese and English project READMEs and a privacy-safe vector preview.
3. Added direct platform download links and newcomer-friendly navigation.
4. Added GitHub Actions for tagged release packaging.
5. Updated release archives to include both languages and the license.
6. Validated Go tests, vet, HTML/SVG parsing, shell syntax and archive contents.

## Architecture invariants

- Project presentation remains documentation-only and separate from the app runtime.
- No credentials, downloaded media, account names or local paths are committed.
- Release artifacts are generated from a tag rather than stored in Git.
- Chinese and English documentation ship together.

## Risks and rollback

- Removing the README presentation additions does not change application behavior.

## Follow-ups

- Configure the maintainer's Git name and email, make the initial commit, create the
  public GitHub repository, push `main`, enable Pages, and tag the first release.
- Add more privacy-safe real screenshots later if desired.
