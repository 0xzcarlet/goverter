-- +goose Up
create extension if not exists pgcrypto;

create schema if not exists app;

-- +goose StatementBegin
create or replace function app.touch_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;
-- +goose StatementEnd

create table if not exists app.user_profiles (
  user_id uuid primary key references auth.users(id) on delete cascade,
  display_name text,
  avatar_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists app.stored_files (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  original_name text not null,
  storage_key text not null unique,
  mime_type text not null,
  size_bytes bigint not null check (size_bytes >= 0),
  checksum_sha256 text,
  created_at timestamptz not null default now()
);

create table if not exists app.conversion_jobs (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  source_file_id uuid not null references app.stored_files(id) on delete cascade,
  target_format text not null check (target_format in ('pdf', 'epub')),
  status text not null default 'queued' check (status in ('queued', 'processing', 'done', 'failed')),
  output_file_id uuid references app.stored_files(id) on delete set null,
  error_message text,
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists conversion_jobs_user_created_idx
  on app.conversion_jobs (user_id, created_at desc);

create index if not exists conversion_jobs_status_created_idx
  on app.conversion_jobs (status, created_at asc);

drop trigger if exists touch_user_profiles_updated_at on app.user_profiles;
create trigger touch_user_profiles_updated_at
before update on app.user_profiles
for each row execute function app.touch_updated_at();

drop trigger if exists touch_conversion_jobs_updated_at on app.conversion_jobs;
create trigger touch_conversion_jobs_updated_at
before update on app.conversion_jobs
for each row execute function app.touch_updated_at();

-- +goose StatementBegin
create or replace function app.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public, auth, app
as $$
begin
  insert into app.user_profiles (user_id, display_name, avatar_url)
  values (
    new.id,
    coalesce(new.raw_user_meta_data->>'full_name', new.raw_user_meta_data->>'name'),
    new.raw_user_meta_data->>'avatar_url'
  )
  on conflict (user_id) do nothing;
  return new;
end;
$$;
-- +goose StatementEnd

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
after insert on auth.users
for each row execute function app.handle_new_user();

-- +goose Down
drop trigger if exists on_auth_user_created on auth.users;
drop function if exists app.handle_new_user();
drop trigger if exists touch_conversion_jobs_updated_at on app.conversion_jobs;
drop trigger if exists touch_user_profiles_updated_at on app.user_profiles;
drop function if exists app.touch_updated_at();
drop table if exists app.conversion_jobs;
drop table if exists app.stored_files;
drop table if exists app.user_profiles;
drop schema if exists app;
