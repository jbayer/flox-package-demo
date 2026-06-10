# Flox Build & Publish Demo

This repo demonstrates the end-to-end Flox workflow for packaging an internal
tool, publishing it to FloxHub, and consuming it from a separate environment —
the pattern a team would use to replace a homegrown "versions directory +
Makefile" tool-management system.

The example package is `datecli`, a small Go CLI that prints the current date
and time.

## Repo layout

```
datecli/    Build environment: Go toolchain, source code, and the
            [build.datecli] manifest build + publish configuration
consumer/   Runtime environment: installs the published jbayer/datecli
            package from FloxHub — no Go toolchain, no source code
```

Each directory is an independent Flox environment (`.flox/`), checked into
git alongside the code it supports.

## Prerequisites

- [Flox](https://flox.dev/get) installed
- An authenticated FloxHub account: `flox auth login`. You can verify the
  logged in status with `flox auth status`.
- This repo cloned with a configured remote (`flox publish` validates that
  the current commit is pushed)

## 1. Build the package

The build is defined in [datecli/.flox/env/manifest.toml](datecli/.flox/env/manifest.toml)
as a **manifest build** — a short bash script that runs inside the activated
environment and copies its deliverables into `$out`:

```toml
[build.datecli]
description = "Example CLI that prints the current date and time"
version = "0.1.0"
command = '''
  mkdir -p $out/bin
  CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o $out/bin/datecli ./cmd/datecli
'''
runtime-packages = ["tzdata"]
```

```bash
cd datecli
flox build datecli
./result-datecli/bin/datecli
# Current date and time: Tue, 09 Jun 2026 17:05:00 PDT
```

Two details worth noticing:

- **`runtime-packages`** trims the package's runtime closure. By default every
  package in `[install]` becomes a runtime dependency of the artifact; here the
  Go toolchain stays build-time only and consumers download just the binary
  plus `tzdata`.
- Flox verifies the claim: if the built binary still referenced the Go
  toolchain (for example, without `-trimpath`), the build would fail with an
  "unexpected dependencies" error rather than silently shipping a bloated
  closure.

## 2. Publish to FloxHub

The target org is set as an environment variable in the build environment
(`[vars]` in the manifest), so the same command works for any developer or CI
runner without hardcoding the org:

```toml
[vars]
FLOX_PUBLISH_ORG = "jbayer"
```

```bash
cd datecli
flox activate -- bash -c 'flox publish -o "$FLOX_PUBLISH_ORG" datecli'
```

`flox publish` requires a clean, pushed git tree, then clones the repo to a
temp location and rebuilds from scratch — proving the package builds from
exactly what is in git, not from local machine state. Binaries are signed and
uploaded to the org's catalog store (Flox-hosted by default; orgs can instead
be associated with a customer-owned S3-compatible bucket), and metadata goes
to FloxHub. Packages published to an org are private to its members.

## 3. Consume the package

A consumer — another team, a CI job, a production image — installs the
published package like any other catalog package. They never see the source
code or the build toolchain:

```bash
cd consumer
flox install jbayer/datecli   # already in this repo's consumer manifest
flox activate -- datecli
# Current date and time: Tue, 09 Jun 2026 17:05:00 PDT
```

The consumer environment's manifest is just:

```toml
[install]
datecli.pkg-path = "jbayer/datecli"

[options]
# Limited to the systems the package has been published for so far.
systems = ["aarch64-darwin", "aarch64-linux"]
```

Version constraints work the same as catalog packages, e.g.
`datecli.version = "^0.1"`.

## 4. Publishing from CI

[.github/workflows/publish.yml](.github/workflows/publish.yml) publishes on
every `v*` tag (or manual dispatch) using the official Flox GitHub Action.
It needs one repository secret:

- `FLOX_FLOXHUB_TOKEN` — a FloxHub auth token, read by the `flox` CLI
  directly from the environment.

Because `flox build` targets the host platform, running the same workflow on
runners of different architectures (e.g. `ubuntu-latest` and `macos-latest`)
publishes the additional platform variants of the same package version — the
manifest needs no changes.

## Notes

- The build here uses the default `sandbox = "off"` (impure) mode. Setting
  `sandbox = "pure"` in the `[build.datecli]` section restricts the build to
  git-tracked files (and blocks network access on Linux) for stronger
  reproducibility guarantees; this stdlib-only Go program builds either way.
- `flox activate` in either directory gives every developer the same
  toolchain on macOS and Linux, x86_64 and ARM — the same environments work
  locally and in CI.
