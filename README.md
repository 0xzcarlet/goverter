# Goverter

Lightweight Go web app for converting **PDF to EPUB** and **EPUB to PDF** from the browser. It uses Supabase for authentication and PostgreSQL, stores files locally, and processes conversions with a small in-process worker powered by Calibre's `ebook-convert`.

## Requirements

- Go `1.25+`
- `make`
- Supabase project with Auth enabled
- PostgreSQL database, preferably Supabase Postgres
- Calibre CLI with `ebook-convert`
- Modern browser

Production deployment also expects:

- Linux VPS/server
- Nginx reverse proxy
- systemd service
- Writable data directory for uploads, outputs, and temporary files
- HTTPS for secure cookies and OAuth redirects

Required environment variables:

- `CSRF_AUTH_KEY`
- `SUPABASE_URL`
- `SUPABASE_ANON_KEY`
- `DATABASE_URL`
- `APP_DAILY_CONVERSION_LIMIT`

Check that the conversion engine is available:

```bash
ebook-convert --version
```

## Installation

Clone the repository:

```bash
git clone https://github.com/your-username/file-converter.git
cd file-converter
```

Create your environment file:

```bash
cp .env.example .env
```

Install Go dependencies:

```bash
go mod tidy
```

Run database migrations:

```bash
make migrate-up
```

Start the development server:

```bash
make dev
```

In `development`, `make dev` first applies pending migrations and then starts the web server.
If `MIGRATION_DATABASE_URL` is unreachable, the migration tool falls back to `DATABASE_URL`.

Open:

```text
http://localhost:8081
```

## Environment Configuration

Do not commit `.env`. It contains secrets and database credentials.

Important fields:

- `APP_ENV`: app environment, usually `development` or `production`.
- `APP_NAME`: app name shown in the UI.
- `APP_BASE_URL`: public app URL, for example `http://localhost:8081`.
- `APP_HOST`: Go server host, default `127.0.0.1`.
- `APP_PORT`: Go server port, default `8081`.
- `APP_COOKIE_DOMAIN`: empty for local development, app domain for production.
- `APP_SECURE_COOKIES`: `false` for local HTTP, `true` for production HTTPS.
- `APP_SESSION_COOKIE_NAME`: session cookie name prefix.
- `APP_DATA_DIR`: root directory for uploads, outputs, and temporary files.
- `APP_MAX_UPLOAD_MB`: max upload size in MB.
- `APP_DAILY_CONVERSION_LIMIT`: daily conversion limit per user.
- `APP_WORKER_POLL_INTERVAL`: worker polling interval.
- `APP_HTTP_READ_TIMEOUT`: request read timeout.
- `APP_HTTP_WRITE_TIMEOUT`: response write timeout.
- `APP_HTTP_IDLE_TIMEOUT`: idle connection timeout.
- `CSRF_AUTH_KEY`: CSRF secret, minimum 32 characters.
- `SUPABASE_URL`: Supabase project URL.
- `SUPABASE_ANON_KEY`: Supabase anon or publishable key.
- `DATABASE_URL`: PostgreSQL connection string for the app runtime.
- `MIGRATION_DATABASE_URL`: PostgreSQL connection string for migrations. If empty, `DATABASE_URL` is used.
- In some Supabase setups, the direct `db.<project-ref>.supabase.co` host may be unreachable from local networks that do not have working IPv6 routing. In that case, the app falls back to `DATABASE_URL` for migrations during local development.

Example development `.env`:

```dotenv
APP_ENV=development
APP_NAME="File Converter"
APP_BASE_URL=http://localhost:8081
APP_HOST=127.0.0.1
APP_PORT=8081
APP_COOKIE_DOMAIN=
APP_SECURE_COOKIES=false
APP_SESSION_COOKIE_NAME=fc_session
APP_DATA_DIR=./var/data
APP_MAX_UPLOAD_MB=25
APP_DAILY_CONVERSION_LIMIT=3
APP_MAX_CONCURRENT_JOBS=1
APP_WORKER_POLL_INTERVAL=3s
APP_HTTP_READ_TIMEOUT=30s
APP_HTTP_WRITE_TIMEOUT=300s
APP_HTTP_IDLE_TIMEOUT=60s

CSRF_AUTH_KEY=replace-with-minimum-32-character-secret

SUPABASE_URL=https://your-project-ref.supabase.co
SUPABASE_ANON_KEY=replace-with-your-supabase-anon-key
DATABASE_URL=postgres://postgres.yourproject:password@aws-0-region.pooler.supabase.com:5432/postgres?sslmode=require
MIGRATION_DATABASE_URL=postgres://postgres:password@db.yourproject.supabase.co:5432/postgres?sslmode=require
```

Production values should use HTTPS and a persistent data directory:

```dotenv
APP_ENV=production
APP_BASE_URL=https://app.example.com
APP_COOKIE_DOMAIN=app.example.com
APP_SECURE_COOKIES=true
APP_DATA_DIR=/srv/file-converter/data
APP_DAILY_CONVERSION_LIMIT=3
```

## Supabase Setup

In the Supabase dashboard:

- Set `Site URL` to your app URL.
- Add local redirect URLs:
  - `http://localhost:8081/auth/callback`
  - `http://localhost:8081/auth/reset`
- Add production redirect URLs:
  - `https://app.example.com/auth/callback`
  - `https://app.example.com/auth/reset`
- Enable Email/Password authentication.
- Enable Google provider if OAuth login is needed.
- Configure custom SMTP for reliable production auth emails.

## Available Commands

```bash
make dev
```

Apply pending migrations in `development`, then run the local web server from `cmd/web`.

```bash
make test
```

Run all Go tests.

```bash
make build
```

Build the web app, migration tool, and smoke check binaries into `bin`.

```bash
make migrate-up
```

Run database migrations.

```bash
make migrate-down
```

Rollback the latest migration.

```bash
make smoke
```

Check database connectivity and `ebook-convert` availability.

```bash
make clean
```

Remove build outputs from `bin`.

## Features

- Email/password registration and login
- Google OAuth via Supabase
- Forgot password and reset password
- Server-side session cookies
- CSRF protection
- Basic security headers
- PDF and EPUB upload
- `pdf -> epub` and `epub -> pdf` conversion
- PostgreSQL-backed conversion queue
- In-process background worker
- Daily per-user conversion quota using `Asia/Jakarta`
- Dashboard with job history and status
- htmx-powered auto-refresh for job status
- Download completed conversion output
- `/healthz` health check
- `/readyz` readiness check
- Goose database migrations
- Nginx and systemd deployment templates

## Roadmap

- [ ] Retry failed jobs from the dashboard
- [ ] Automatic cleanup for old files
- [ ] Admin dashboard for users and jobs
- [ ] Additional file formats beyond PDF and EPUB
- [ ] Multi-format file conversion for documents, ebooks, images, and archives
- [ ] Email notification when conversion is complete
- [ ] Object storage support, such as Supabase Storage or S3-compatible storage
- [ ] Stronger rate limiting
- [ ] Metrics, logs, and error observability
- [ ] More detailed realtime conversion progress
- [ ] User activity audit trail

## Closing

`file-converter` is designed as a focused, small-footprint file conversion service for personal tools, internal utilities, or lightweight VPS deployments.

For production, use secure environment values, HTTPS, valid Supabase redirect URLs, applied database migrations, and an available `ebook-convert` binary on the server.

## License

This project is licensed under the [MIT License](LICENSE).
