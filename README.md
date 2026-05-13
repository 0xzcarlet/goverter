# file-converter

Native Go scaffold for a small-VPS PDF/EPUB converter that sits behind host-level Nginx and uses Supabase for auth plus Postgres.

## Stack

- Go web app on `127.0.0.1:8081`
- `chi` router with `slog`, CSRF, health checks, and secure headers
- Server-rendered UI using `templ` runtime components
- Supabase Auth for email/password, Google OAuth, forgot password, and reset password
- Supabase Postgres via `pgx/v5`
- `goose` migrations
- Local file storage under `/srv/file-converter/data`
- In-process worker with concurrency `1`
- Daily per-user conversion quota driven by `APP_DAILY_CONVERSION_LIMIT`
- `ebook-convert` from Calibre for `pdf <-> epub`
- Production deploy via `systemd` plus host `Nginx`

## Quick start

1. Copy `.env.example` to `.env` and fill in Supabase values.
   Set `APP_DAILY_CONVERSION_LIMIT` explicitly; startup fails if it is missing or below `1`.
2. Run `go mod tidy`.
3. Run `make migrate-up`.
4. Ensure `ebook-convert` is installed locally.
5. Run `make dev`.
6. Open `http://localhost:8081`.

## Required Supabase setup

- Set `Site URL` to your app base URL.
- Add `http://localhost:8081/auth/callback` and `https://app.example.com/auth/callback` to redirect URLs.
- Add `http://localhost:8081/auth/reset` and `https://app.example.com/auth/reset` to redirect URLs.
- Configure Google provider in Supabase Auth.
- Configure custom SMTP in Supabase Auth.
- Apply the SQL migrations from `db/migrations`.

## Production layout

- App binary: `/srv/file-converter/bin/file-converter`
- Data root: `/srv/file-converter/data`
- Env file: `/etc/file-converter.env`
- Nginx site template: `deploy/nginx/file-converter.conf`
- systemd unit: `deploy/systemd/file-converter.service`
- Example env values: `deploy/env/file-converter.env.example`

## Notes

- `web/assets/js/htmx.min.js` is pinned to HTMX `2.0.10` and served locally by the Go app.
- Google OAuth callback handling is wired through Supabase redirects and server-set secure cookies.
- The worker intentionally processes only one queued conversion at a time to fit a 1 GiB VPS alongside Ghost.
