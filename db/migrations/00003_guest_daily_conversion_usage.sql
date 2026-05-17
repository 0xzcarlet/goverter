-- +goose Up
create table if not exists app.guest_daily_conversion_usage (
  guest_token text not null,
  quota_date date not null,
  reserved_count integer not null default 0 check (reserved_count >= 0),
  completed_count integer not null default 0 check (completed_count >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (guest_token, quota_date)
);

create index if not exists guest_daily_conversion_usage_quota_date_idx
  on app.guest_daily_conversion_usage (quota_date);

drop trigger if exists touch_guest_daily_conversion_usage_updated_at on app.guest_daily_conversion_usage;
create trigger touch_guest_daily_conversion_usage_updated_at
before update on app.guest_daily_conversion_usage
for each row execute function app.touch_updated_at();

-- +goose Down
drop trigger if exists touch_guest_daily_conversion_usage_updated_at on app.guest_daily_conversion_usage;
drop table if exists app.guest_daily_conversion_usage;
