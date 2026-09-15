# Raccodown

Raccodown is a self-hosted Markdown/plain-text note app: a CodeMirror 6 editor with a formatting
toolbar and a live preview, backed by a Go REST API, with a UI available in five languages. Like
this author's other Go projects (e.g. raccounting), it's **one binary**: the frontend is embedded
into it via `go:embed` and served from the same port as the API — no separate frontend process, no
CORS to configure in production.

It's single-user, cookie-authenticated — there's no multi-tenant model, just one signed-in account
and their notes — with a demo account seeded on first run so a fresh checkout is loggable-into
without a registration flow.

Notes, users and sessions persist in a database — MySQL/MariaDB by default, Postgres, or SQLite with
no server to run at all — opened and migrated automatically on startup.

## Architecture

The backend follows a hexagonal ("ports & adapters") architecture: business logic is organized as
one explicit **use case** per action (e.g. `note/create`, `auth/login`), each with its own
`Input`/`Output`/`Execute`. Use cases depend only on an interface declared in `internal/port` —
never on a concrete storage implementation or on `net/http`. The concrete implementation
("adapter") is plugged in once, at the composition root.

The frontend is **buildless** — no bundler, no `package.json`, no `node_modules` — served straight
from `web/` via `go:embed`, the same philosophy as raccounting's frontend. It differs from
raccounting's in one respect: instead of classic `<script>` tags and a `window.App` global, it's one
native ES module graph (own files import each other with plain relative paths; third-party
packages are pinned to exact CDN URLs in `web/index.html`'s import map). That's not a style
preference — CodeMirror 6 has no single-file browser build, only real ES modules, so mixing that
with script-tag globals for everything else would mean two loading styles in one app for no reason.

```
cmd/raccodown/             thin entry point; delegates everything to internal/app
internal/
  app/                      composition root: loads config, wires repositories, services, use cases
                            and the HTTP server together, seeds the demo notes/account on first run,
                            runs the session-cleanup sweep, starts serving
  config/                   env var loading & validation (app.go / log.go), fails fast on anything
                            malformed
  domain/
    entity/                 Note, User, PublicUser, UserSettings, Session — the core types; Note,
                            PublicUser and UserSettings define their own MarshalJSON/UnmarshalJSON
                            wire shape
    error/                  typed errors — ValidationError, NotFoundError, ConflictError,
                            UnauthorizedError — mapped to HTTP status codes in one place
                            (internal/adapter/http/errors.go)
    usecase/
      note/                 one directory per action: create, get, list, update, delete
      tag/                  list — tags aren't stored on their own, they're derived from every
                            note's content (see usecase.ExtractTags)
      auth/                 login, logout, authenticate (session token -> user, used by the
                            requireAuth middleware), updatecredentials
      settings/             get, update — the signed-in user's saved UI settings (language, theme)
    service/
      passwordhasher/       bcrypt
      tokengenerator/       random session tokens
  port/                     the interfaces use cases depend on (NoteRepository, UserRepository,
                            SessionRepository, PasswordHasher, TokenGenerator), plus
                            mockgen-generated mocks for testing
  repository/               one package per entity (note, user, session): converts between domain
                            entities and storage/model row shapes and turns raw storage errors into
                            typed domain errors — depends on a small Storage interface of its own,
                            satisfied by whichever driver adapter is configured (see Persistence
                            below)
  storage/
    model/                  the shape of a database row for each table, with a ToEntity() converter;
                            kept separate from the domain entities so a column type (e.g. the users.
                            settings JSON blob) never leaks into domain/entity
    error/                  portable sentinel errors with no equivalent in database/sql (e.g.
                            UniqueViolationError), which each driver adapter normalizes its own
                            driver errors into
  adapter/
    mysql/                  the MySQL/MariaDB driver adapter (the default) — connection pool setup,
                            goose migrations (migrations/*.sql, embedded into the binary), and the
                            actual queries, one file per table group (note.go, user.go, session.go)
    postgres/               the Postgres driver adapter, same shape as mysql/ above (its own dialect:
                            "$1, $2, ..." placeholders, RETURNING id for generated ids), selected via
                            DB_DRIVER=postgres
    sqlite/                 the SQLite driver adapter, same shape as mysql/ above, selected via
                            DB_DRIVER=sqlite
    http/                   the driving adapter: net/http handlers, routing, cookies, JSON
                            encoding/decoding, and use-case-error -> HTTP-status mapping

web/
  webassets.go              go:embed for the frontend (must live under web/, since go:embed can't
                            reach outside the directory of the file that declares it)
  index.html                import map for every vendored package (Vue, CodeMirror, markdown-it —
                            see js/vendor/README.md) + the mount point
  css/style.css             theme (light/dark via prefers-color-scheme, or pinned explicitly — see
                            data/theme.js) + layout
  js/
    data/                   api.js (fetch wrapper), markdown.js (configured markdown-it instance,
                            with the markdown-it-multimd-table plugin for GFM-style tables),
                            markdown-commands.js (toolbar formatting commands for CodeMirror),
                            theme.js (light/dark preference), i18n.js (t() + the five translation
                            tables — see Internationalization below); both are just the local
                            mechanism — store/auth.js layers the saved-server-side settings on top
    store/                  notes.js, auth.js — Vue.reactive() singletons (no Pinia; there's no
                            build step to bring in a package like that); auth.js also syncs the
                            signed-in user's language/theme with GET/PATCH /api/v1/settings
    components/             login-view.js, note-list.js (sidebar), editor.js (CodeMirror +
                            formatting toolbar), preview.js (rendered Markdown), note-view.js
                            (toolbar + panes + conflict banner), credentials-modal.js (change
                            username/password), delete-note-modal.js (confirm before delete)
    app.js                  bootstraps the Vue app: login-view until /api/v1/auth/me resolves,
                            the note UI after

build/app/                  make build output (gitignored)
build/docker/Dockerfile     minimal Alpine image that copies in a pre-built static binary
docker-compose.yml          a local MariaDB container for development
Makefile                    run / build / test / lint / docker-* targets
```

### Note updates and conflicts

A client must send back the `Checksum` it last read (as the `If-Match` header) for `PUT
/api/v1/notes/{id}` to succeed unconditionally. If the note changed in the meantime, the request
comes back as `409 Conflict` with the note's current state in the body, so the client can offer a
merge instead of just failing — see `internal/domain/usecase/note/update.ConflictError` and
`adapter/http.handleUpdateNote`.

### Persistence

The database is pluggable, chosen at startup by `DB_DRIVER` (see `internal/app.openStorage`): mysql
(the default, `github.com/go-sql-driver/mysql`, works against MariaDB too), postgres
(`github.com/jackc/pgx/v5`), or sqlite (`modernc.org/sqlite`, a pure-Go driver — no CGO, no system
SQLite required to build or run — good for a quick local run with nothing else to install).
Whichever is picked, startup makes sure the target database exists (best-effort — `CREATE DATABASE
IF NOT EXISTS` for mysql, a `pg_database` existence check plus `CREATE DATABASE` for postgres, the
parent directory for sqlite), opens the connection pool, and applies any pending migrations
(`github.com/pressly/goose/v3`, schema in `internal/adapter/<driver>/migrations/`, embedded into the
binary via `go:embed`) — a fresh checkout just works, with no separate setup step beyond having a
database to point at.

Every repository depends only on a small `Storage` interface of its own (`internal/repository/*`),
and `internal/app/storage.go` is the one place that asks a single adapter — mysql, postgres or
sqlite — to satisfy all of them at once, so a driver missing an operation fails to compile there
rather than anywhere downstream. SQLite additionally doesn't tolerate concurrent writers, so its
pool is capped at a single connection (`internal/adapter/sqlite.Open`) regardless of
`DB_MAX_OPEN_CONNS` — for a single-user app talking to its own local file, that's not a practical
bottleneck; MySQL and Postgres use the configured pool settings as-is. Both `entity.Note.ID` and
`entity.User.ID` are `uint64`, assigned by the database's own autoincrement
(`Storage.insertReturningID` for users, and `CreateNote`'s own id-generating insert for notes, since
it needs to run inside the same transaction as the tag inserts — `LastInsertId()` on mysql/sqlite,
a `RETURNING id` clause on postgres, which has no `LastInsertId()` equivalent) — a note's id is
never known before its first write lands, unlike the checksum, which is computed from content
before the insert.

Tags are normalized: a `tags` table (one row per distinct tag name) and a `note_tags` join table
(`note_id`, `tag_name`, both `ON DELETE CASCADE`) — indexed on `note_tags.tag_name` for the
tag -> notes direction, with `note_tags`'s own primary key `(note_id, tag_name)` covering the
notes -> tags direction. `CreateNote`/`UpdateNote` upsert each tag (`INSERT ... ON CONFLICT DO
NOTHING` on sqlite and postgres, `INSERT IGNORE` on mysql, which predates standard `ON CONFLICT`
support) and rewrite the note's links in one transaction (`note.go`'s `setNoteTags`), so a tag row
is never duplicated and a note's tag set is always replaced atomically, not diffed. A tag left
linked to no note is pruned in the same transaction (`pruneUnusedTags`), so the `tags` table never
accumulates dead rows.

### Authentication

A bcrypt-hashed password, a random session token, and an httpOnly `SameSite=Lax` cookie — every
route but `POST /api/v1/auth/login` requires a valid session (see `adapter/http.requireAuth`).
Sessions last 30 days and are swept for expiry once an hour in the background
(`internal/app/sessions_cleanup.go`); an expired token presented before that sweep runs is caught
and deleted lazily by the `authenticate` use case itself. There's no registration flow or
per-account data isolation to reason about, since the whole app is one user's notes; credentials
can be changed via `PATCH /api/v1/auth/credentials`, which always re-confirms the current password
first — the sidebar's user menu has a "Change username and password" modal for this
(`web/js/components/credentials-modal.js`).

## Getting started

### Requirements

- Go 1.26+ to build from source (a prebuilt binary needs nothing but a database).
- One of:
  - MySQL or MariaDB, with a user that can create the database (or a database created ahead of
    time) — the default;
  - PostgreSQL, likewise;
  - or nothing at all — SQLite just needs a writable path for its database file.
- Docker, only if you want to run the bundled local MariaDB via `docker-compose.yml`.

### Configuration

Copy `.env.example` to `.env` and adjust it — both `docker compose` (for the local MariaDB
container) and the app itself (via [godotenv](https://github.com/joho/godotenv)) read it
automatically. Real environment variables always take priority over `.env`.

| Variable | Required | Default | Notes |
|---|---|---|---|
| `DB_DRIVER` | no | `mysql` | `mysql`, `postgres` or `sqlite` |
| `DB_HOST` | mysql/postgres only | — | |
| `DB_PORT` | mysql/postgres only | — | typically `3306` / `5432` |
| `DB_USER` | mysql/postgres only | — | |
| `DB_PASSWORD` | no | empty | read raw, not trimmed — a password may legitimately contain spaces |
| `DB_NAME` | mysql/postgres only | — | |
| `DB_SQLITE_PATH` | sqlite only | `data/raccodown.db` | database file, created (with parent dirs) on first run |
| `DB_MAX_OPEN_CONNS` | no | `10` | ignored for sqlite (always 1 — no concurrent writers) |
| `DB_MAX_IDLE_CONNS` | no | `5` | ignored for sqlite |
| `DB_CONN_MAX_LIFETIME` | no | `5m` | Go duration syntax |
| `DB_CONN_TIMEOUT` | no | `5s` | initial TCP dial timeout, mysql only (Postgres has no equivalent setting — see `postgres.Config`) |
| `PORT` | no | `8080` | HTTP port the server listens on |
| `COOKIE_SECURE` | no | `false` | set `true` in production (HTTPS) so the session cookie requires TLS; a startup warning is logged while this is `false` |
| `DEMO_USERNAME` / `DEMO_PASSWORD` | no | `user` / `pass` | the single account seeded on first run, only while no user exists yet |
| `LOG_CONSOLE_LEVEL` | no | `warn` | `debug` / `info` / `warn` / `error`, plain text to stdout |
| `LOG_FILE_PATH` | no | unset | JSON logs to a file, for a log aggregator; off unless set (nothing rotates it) |
| `LOG_FILE_LEVEL` | no | `info` | only used if `LOG_FILE_PATH` is set |

A value that's set but malformed (`PORT=http`, `COOKIE_SECURE=yes`) fails startup immediately with a
specific error, rather than silently falling back to a default.

### Run it

```bash
cp .env.example .env
docker compose up -d   # starts a local MariaDB container for development
go run ./cmd/raccodown
```

Open `http://localhost:8080` and log in with `DEMO_USERNAME`/`DEMO_PASSWORD` (`user` / `pass` by
default). The server seeds that account, plus one welcome note per supported UI language, the first
time it runs against an empty database (see [Persistence](#persistence)).

To run against Postgres instead:

```bash
DB_DRIVER=postgres DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=raccodown DB_PASSWORD=raccodown \
  DB_NAME=raccodown go run ./cmd/raccodown
```

To run against SQLite instead — no Docker, no server needed:

```bash
DB_DRIVER=sqlite DB_SQLITE_PATH=./data/raccodown.db go run ./cmd/raccodown
```

Deleting the SQLite file (or the MySQL/Postgres database) resets to a fresh install.

If the configured MySQL/Postgres user can't create databases, create it by hand first:

```sql
-- MySQL/MariaDB
CREATE DATABASE raccodown CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

```sql
-- Postgres
CREATE DATABASE raccodown;
```

**Editing the frontend:** `web/` is embedded at *compile* time, so a change under `web/` needs the
process restarted (`go run` again) to show up — there's no dev-server hot reload the way a
Vite/webpack setup would have. This is the same tradeoff raccounting's embedded frontend makes.

### Building & running the binary

```bash
go build -o raccodown ./cmd/raccodown
./raccodown
```

The binary already contains the entire frontend (`web/index.html`, `css/`, `js/`) via `go:embed` —
nothing else needs to ship alongside it.

`make build` cross-compiles a static (`CGO_ENABLED=0`) Linux/amd64 binary into `build/app/`, ready
to be copied into the Alpine-based image at `build/docker/Dockerfile`.

### Docker image

```bash
make build                                  # produces build/app/raccodown
docker build -f build/docker/Dockerfile -t raccodown:latest .
docker run -p 8080:8080 --env-file .env raccodown:latest
```

`build/docker/Dockerfile` is a minimal `alpine:3.20` image (plus `ca-certificates`) that just copies
in the pre-built binary — the actual compilation happens on the host via `make build`, not inside
the image.

### Useful `make` targets

| Target | Does |
|---|---|
| `make run` | `go run ./cmd/raccodown` |
| `make build` | cross-compile a static binary into `build/app/raccodown` |
| `make test` | `go test ./...` |
| `make fmt` / `make vet` | `go fmt` / `go vet` |
| `make lint` | run `golangci-lint` (fetched into `./bin` by `make install-deps`) |
| `make docker-up` / `docker-down` / `docker-restart` / `docker-logs` / `docker-ps` | manage the local MariaDB container from `docker-compose.yml` |

Run `make help` for the full list.

## HTTP API

All responses are JSON; errors are `{"error": "<message>"}` with an appropriate HTTP status code
(400 validation / 401 unauthorized / 404 not found / 409 conflict / 500 internal). Every route but
`POST /api/v1/auth/login` requires a valid session cookie and returns `401` without one.

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/health` | liveness check — not behind auth |
| POST | `/api/v1/auth/login` | `{username, password}` -> sets the session cookie, returns the user |
| POST | `/api/v1/auth/logout` | clears the session |
| GET | `/api/v1/auth/me` | current user from the session cookie (used to restore a session after a page reload) |
| PATCH | `/api/v1/auth/credentials` | change username/password, re-confirming the current password |
| GET | `/api/v1/notes` | list notes; `?q=` full-text filter, `?tag=` tag filter |
| POST | `/api/v1/notes` | create a note |
| GET | `/api/v1/notes/{id}` | fetch one note |
| PUT | `/api/v1/notes/{id}` | update a note; `If-Match: <checksum>` guards against conflicting writes |
| DELETE | `/api/v1/notes/{id}` | delete a note |
| GET | `/api/v1/notes/export` | download every note as one `.md` file each, bundled into a zip |
| GET | `/api/v1/tags` | every distinct tag currently in use |
| GET | `/api/v1/settings` | the signed-in user's saved UI settings (language, theme) |
| PATCH | `/api/v1/settings` | overwrite both settings fields at once — a client always echoes back the one it isn't changing |

## Internationalization

The UI supports Russian, English, Spanish, German and French, by analogy with raccounting's
approach: Russian is the language the frontend is actually written in — every UI string is a
Russian literal at its call site, passed through `t()` — and the other four languages are
exact-match translation tables in `web/js/data/i18n.js`, keyed by that Russian source string. A
string missing from a table simply falls back to its Russian source text rather than breaking. A
handful of entries carry `{placeholders}` (e.g. `'Удалить заметку «{title}»?'`) for
runtime-interpolated values.

The chosen language and theme are both saved server-side, by analogy with raccounting: `User`
carries a `Settings` field (`Language`, `Theme`), stored as a JSON blob in the `users.settings`
column (see `internal/storage/model.UserSettings`) and exposed over `GET`/`PATCH /api/v1/settings`.
`web/js/data/i18n.js` and `web/js/data/theme.js` are still the local mechanism — translate/apply +
remember in `localStorage` — but `web/js/store/auth.js` layers the backend on top: a fresh account
has no saved preference yet, so the frontend falls back to browser/OS detection until one is
explicitly chosen; logging in applies whatever the account has saved (overriding local detection,
the same as raccounting); and changing either setting in the sidebar footer round-trips through
`PATCH /api/v1/settings` before the UI updates, rather than switching optimistically. The first
login ever also saves whatever language the browser detected, if the account doesn't have one on
record yet — so it doesn't need choosing twice.

Switching languages reaches everything, including note titles created from then on (e.g. "New note"
vs "Новая заметка") and the CodeMirror empty-note placeholder — though not any language-independent
user data itself (note content, existing titles) or error messages that come back from the Go
backend, which are always in English.

## Testing

Every use case, repository, and HTTP handler has unit test coverage: use cases and repositories are
tested against `go.uber.org/mock`-generated mocks of the `internal/port` and `internal/repository/*.
Storage` interfaces (`go generate ./...` regenerates them). The three driver adapters differ in how
they're tested: `internal/adapter/sqlite` runs against a real in-memory SQLite database (migrated
with goose, same as production) so the actual SQL is what's under test; `internal/adapter/mysql` and
`internal/adapter/postgres` have no in-memory equivalent to run against, so they're tested with
`DATA-DOG/go-sqlmock` instead — each query is asserted against its exact expected SQL and arguments
(mysql's `?` placeholders vs. postgres's `$1, $2, ...`), by analogy with raccounting's own adapter
tests. `stretchr/testify` provides assertions and `brianvoe/gofakeit` generates realistic test data
where it's useful. Run everything with:

```bash
go test ./...
# or
make test
```

There's no frontend test suite yet — it's plain JS with no build step to run one through.

## License

MIT — see [LICENSE](LICENSE).
