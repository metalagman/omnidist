# Release runbook

Registry publication is irreversible and not transactional across packages or backends. Omnidist preflights every selected backend before aggregate upload begins, but a registry rejection, race, or network failure during upload can still produce a partial release.

## Credentials

| Backend | Token mode | Trusted mode |
| --- | --- | --- |
| npm | `NPM_PUBLISH_TOKEN`; optional CLI `--registry`, `--tag`, `--otp` | Set `publish-auth: trusted`, provide `repository-url`, configure every npm package as a trusted publisher, and grant `id-token: write`. |
| uv | `UV_PUBLISH_TOKEN` or `uv publish --token`; optional `--publish-url` | Not implemented; use token authentication. |
| gem | `GEM_HOST_API_KEY` or `RUBYGEMS_API_KEY`; optional `GEM_HOST_OTP_CODE` / `--otp` | Set `publish-auth: trusted`, use rubygems.org, configure the repository/workflow on RubyGems, and grant `id-token: write`. |

Dry-run validates tooling, staged artifacts, and known version policy without requiring live credentials or npm `whoami`. Trusted-publishing preflight validates local configuration only; it cannot prove registry-side OIDC eligibility.

Never commit tokens or place them in generated workflow YAML. The CI generator references GitHub secrets only for enabled token-auth backends.

## First-release checklist

1. Confirm package names are available and registry accounts have permission to publish them. When npm `aliases` or `platform-package` use different scopes from `package`, verify ownership of every identity; public scoped packages require `access: public`.
2. Review `.omnidist/omnidist.yaml`, especially the present `distributions` sections, package names, repository URLs, access, authentication modes, and target matrix.
3. For npm trusted mode, run `omnidist npm trust` to inspect setup commands for every meta package and platform package; use `--apply` only after reviewing them. For a previously unpublished name, review whether `--allow-stage-publish` is needed before the first release.
4. Configure CI secrets or trusted publishers for every selected backend.
5. Generate and review CI without writing first: `omnidist ci --dry-run`.
6. Build, stage, verify, and exercise publish preflight:

   ```bash
   omnidist build
   omnidist stage
   omnidist verify
   omnidist publish --dry-run
   ```

7. Inspect staged package names and versions. All must represent the same release.
8. Publish from one controlled environment. For tag-driven CI, create and push an exact SemVer tag only after credentials and workflow settings are ready.

## What preflight checks

Before aggregate `publish` uploads anything, Omnidist checks every selected backend:

- required executable availability;
- staged artifact presence and structure;
- version validity and locally known registry policy, including PyPI local-version rejection;
- token presence, or locally verifiable trusted-publishing configuration;
- npm token authentication through `npm whoami` outside dry-run.

If any selected backend fails, the command reports all preflight failures and attempts zero uploads. Direct `npm publish`, `uv publish`, and `gem publish` subcommands perform their own backend preflight too.

Preflight cannot reserve names or versions, attest remote OIDC configuration, prevent concurrent publication, or roll back an accepted upload.

## Publishing and progress

Aggregate upload order is npm, then uv, then gem, independent of YAML order. npm publishes each unique platform package first, then the primary package followed by aliases in configured order; gems are sorted deterministically. Progress messages name completed units/backends. Preserve the complete log when a failure occurs.

Use backend commands when options differ by registry:

```bash
omnidist npm publish --tag latest --registry https://registry.npmjs.org
omnidist uv publish --publish-url https://test.pypi.org/legacy/
omnidist gem publish --host https://rubygems.org
```

These commands are also the recovery path for a configured backend excluded by a legacy selector. A direct command cannot run a backend whose section is absent.

## Partial-release recovery

Do not delete tags, retag a different build with the same version, or assume retry is an atomic rollback.

1. Stop automated retries and retain the publish log.
2. Inspect each registry to identify exactly which package/version units exist. A partial npm release may have some platform packages or meta packages but not others; inspect the primary name, every alias namespace, and the `platform-package` namespace. RubyGems may have only some platforms.
3. Compare accepted artifacts with the locally verified staged artifacts and checksums.
4. Correct the local or remote cause without rebuilding the same version differently.
5. Retry only the missing backend with its backend-specific publish command. Registry clients commonly reject already-existing versions, so use the progress log and registry state to avoid blindly replaying accepted units.
6. If the registry cannot resume the missing units safely, publish a new patch version across the intended backends and communicate the incomplete version.
7. Verify consumer installation on representative platforms after recovery.

Omnidist does not delete or overwrite registry releases automatically.

## CI releases

`omnidist ci` generates `.github/workflows/omnidist-release.yml`. It uses only the resolved selected backends, stages and verifies with an explicit `--only` list, creates backend publish jobs, and creates a separate GitHub Release job for binaries/checksums.

Separate publish jobs can fail independently after another registry succeeds. Treat the GitHub Actions run as a coordinated workflow, not a cross-registry transaction, and use the recovery procedure above.
