# Plan: Artemis Go + Flox Build

## 1. Goals
- A Go binary (`artemis`) that renders Go templates from `assets/templates/`
  into an output directory and writes a `metadata.json` describing the build.
- Flox orchestrates: provides Go toolchain for dev, and a `[build.*]` stanza
  that (a) compiles the binary and (b) executes it exactly once to populate
  `$out`.

## 2. Proposed layout
```
Artemis/
├── .flox/env/manifest.toml        # flox env + build definitions
├── cmd/artemis/main.go            # CLI entrypoint (flags, wiring)
├── internal/
│   ├── render/render.go           # text/template walker
│   ├── hash/hash.go               # deterministic dir hash
│   ├── gitinfo/gitinfo.go         # git SHA / branch / dirty
│   └── metadata/metadata.go       # metadata struct + writer
├── assets/
│   └── templates/                 # <-- fixed path, input gotemplates
│       ├── config.yaml.tmpl
│       └── ...
├── go.mod / go.sum
└── README.md
```

Rationale: `assets/templates/` keeps room for sibling non-template assets
under `assets/` later.

## 3. Go binary design (`cmd/artemis`)

CLI flags:
- `--templates-dir` (default `./assets/templates`)
- `--output-dir` (default `$out` if set, else `./build`)
- `--initiator` (default: best-effort — first non-empty of
  `FLOX_BUILD_INITIATOR`, `GITHUB_ACTOR`, `CI_JOB_NAME`, `USER`, then
  `unknown`)
- `--version` (ldflags-injected)

Execution pipeline:
1. **Validate inputs** — templates dir exists, output dir is empty or
   creatable.
2. **Walk templates dir** — for every file ending in `.tmpl`, parse with
   `text/template`, execute against a context struct (§4), write to
   `output/<relpath minus .tmpl>`. Non-`.tmpl` files are **copied verbatim**
   so static assets work. Preserve file modes.
3. **Hash templates dir** — canonical SHA-256 over a sorted list of
   `(relpath, mode, sha256(content))` tuples. Emit as `sha256:...`.
   Aggregate only for now; a `// TODO` in the metadata package tracks
   expanding to per-file hashes.
4. **Collect git info** — try `git rev-parse HEAD`,
   `git rev-parse --abbrev-ref HEAD`, and `git status --porcelain` (for dirty
   flag). Fall back to env vars (`GIT_COMMIT`, `GITHUB_SHA`) when `.git` is
   absent.
5. **Write metadata** — `output/metadata.json` (see §5). Written last so it
   is *not* included in the source hash.

Exit non-zero on any template parse/render failure; fail-fast rather than
partial output.

## 4. Template context
```go
type TemplateCtx struct {
    GitCommit  string
    GitBranch  string
    GitDirty   bool
    Timestamp  time.Time           // RFC3339, UTC
    Initiator  string              // best-effort env-var lookup
    Version    string              // ldflags
    Env        map[string]string   // allowlisted env vars only
}
```
Register helper funcs via `template.FuncMap`: `upper`, `lower`, `default`,
`toYaml`, `toJson`, `env "NAME"`.

## 5. `metadata.json` schema
```json
{
  "schemaVersion": 1,
  "tool":      { "name": "artemis", "version": "0.1.0" },
  "source":    { "templatesDir": "assets/templates", "hash": "sha256:..." },
  "git":       { "commit": "abcdef...", "branch": "main", "dirty": false },
  "build":     { "timestamp": "2026-04-15T12:34:56Z", "initiator": "flox-build:alice" }
}
```
- `source.hash` is a single aggregate over all inputs, excluding
  `metadata.json` itself.
- `// TODO(metadata): add outputs[]{path,sha256} with per-file hashes` lives
  in code so it surfaces in review.

## 6. Flox manifest (`.flox/env/manifest.toml`)

```toml
version = 1

[install]
go.pkg-path   = "go"
git.pkg-path  = "git"

[vars]
ARTEMIS_TEMPLATES_DIR = "assets/templates"

[build.artemis]
description  = "Artemis template renderer binary"
command = '''
  export GOCACHE="$FLOX_BUILD_CACHE/go-build"
  export GOMODCACHE="$FLOX_BUILD_CACHE/go-mod"
  mkdir -p "$out/bin"
  go build -trimpath \
    -ldflags "-X main.version=$(git rev-parse --short HEAD 2>/dev/null || echo dev)" \
    -o "$out/bin/artemis" ./cmd/artemis
'''

[build.artemis-output]
description  = "Rendered templates + metadata.json"
runtime-packages = ["artemis", "git"]
command = '''
  mkdir -p "$out"
  artemis \
    --templates-dir "./$ARTEMIS_TEMPLATES_DIR" \
    --output-dir    "$out" \
    --initiator     "${FLOX_BUILD_INITIATOR:-${USER:-flox}}"
'''
sandbox = "off"   # needed so git + env can flow into metadata
```

Notes:
- Two `[build.*]` targets: `artemis` (pure-ish Go compile) and
  `artemis-output` (depends on `artemis`, runs it). Separation keeps the
  binary cacheable.
- `sandbox = "off"` on the output step is deliberate: git SHA and initiator
  are impure inputs. Documented in README.

## 7. Dev workflow
- `flox activate` → shell with Go + git.
- `go run ./cmd/artemis --templates-dir assets/templates --output-dir build`
  for fast iteration.
- `flox build artemis-output` for the sealed build; result lands in
  `result-artemis-output/`.

## 8. Implementation order (commits)
1. **Scaffolding** — `go.mod`, `cmd/artemis/main.go` stub,
   `assets/templates/example.txt.tmpl`, empty `internal/*`.
2. **Renderer** — `internal/render` + unit tests (golden files).
3. **Hasher** — `internal/hash` + unit test proving order-independence.
4. **Git info + metadata** — `internal/gitinfo`, `internal/metadata` (with
   the per-file-hash TODO), unit tests against a temp git repo.
5. **Wire CLI** — flags, pipeline, error handling.
6. **Flox manifest** — `.flox/env/manifest.toml` with the two build targets.
7. **README** — usage, impurity note, how to add new templates.
8. **Smoke test** — run `flox build artemis-output` end-to-end, verify
   `metadata.json`.

## 9. Resolved questions
- **9.1 Templates path** — `assets/templates/`.
- **9.2 Non-`.tmpl` files** — copied verbatim.
- **9.3 Initiator identity** — best-effort env-var lookup.
- **9.4 Reproducibility / sandbox** — `sandbox = "off"` on the output step is
  accepted.
- **9.5 Output hashing** — aggregate source hash only; `TODO` left for
  per-file hashes.
