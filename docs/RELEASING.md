# Production release runbook

## Release policy

- Canonical repository: `JustinNguyen9979/ainovel-cli`.
- Release source must be a protected commit on `main`.
- Package and tag use the same SemVer version: `package.json` `0.8.0` → tag `v0.8.0`.
- Docker is not part of distribution. Users install through npm or use native GitHub Release archives.
- Supported native targets: Linux x64/arm64, macOS Intel/Apple Silicon, Windows x64/arm64.
- The npm package is Apache-2.0 and contains only the launcher, README and license. The native binary is downloaded from the matching public GitHub Release.

## Before tagging

1. Merge the PR into protected `main`.
2. Confirm `package.json` and `package-lock.json` have the intended same version.
3. Confirm the version has never been published to npm and the Git tag does not exist.
4. Confirm all required CI checks pass:
   - formatting, module verification, vet and tests;
   - race tests;
   - total Go line coverage >= 80%;
   - benchmark regression <= 5%;
   - govulncheck, OSV, Gitleaks, npm audit and Apache license check;
   - native test/build matrix and six-target cross-build;
   - package allowlist and packed npm tarball checks.
5. Confirm the `production` environment has required reviewers.
6. Confirm `NPM_TOKEN` is configured as an environment-scoped Actions secret. Do not put it in source, `.npmrc`, workflow logs or a regular repository file.

## Release

```bash
git tag v0.8.0
git push origin v0.8.0
```

The release workflow then runs preflight, calls the reusable quality workflow, builds Go artifacts once, validates checksums, packs the npm tarball, creates provenance attestations, waits for production approval, uploads the GitHub Release assets, and publishes the exact npm tarball.

Do not rerun a release with a moved tag. Protected tags must reject force updates. A rerun of the same immutable tag may continue missing promotion steps, but must fail if an existing asset or npm version has a different digest.

## npm Trusted Publishing migration

The current production path uses `NPM_TOKEN`. To migrate:

1. Create the npm package under the canonical npm account.
2. Configure npm Trusted Publishing for repository `JustinNguyen9979/ainovel-cli`, the exact release workflow file, and the `production` environment.
3. Test on a new prerelease version.
4. Remove `NPM_TOKEN` and `NODE_AUTH_TOKEN` from the publish job.
5. Keep `id-token: write` only on the publish job and publish with npm provenance.

## Failure and rollback

- npm versions and Git tags are immutable. Never overwrite or move them.
- If publication fails, inspect the workflow's per-step state and rerun the same tag only when the existing artifacts match their expected checksums.
- If an npm version is defective, deprecate it and publish a corrected patch version. Do not unpublish as a routine rollback.
- If a GitHub Release asset is defective, keep the run and digest evidence, then remove/replace only under maintainer approval and never silently clobber a mismatched asset.
- `latest` is a convenience channel; versioned installs remain the rollback mechanism.

## Post-release smoke

From a clean temporary directory, verify the registry package and command:

```bash
npm install --global ainovel-cli-vn@0.8.0
ainovel-cli-vn --version
npx --yes ainovel-cli-vn@0.8.0 --version
```

Repeat on each supported OS/CPU target. Confirm the launcher downloaded the matching GitHub Release archive and that the checksum matches the release manifest.
