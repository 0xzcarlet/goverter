-- +goose Up
create table if not exists app.daily_conversion_usage (
  user_id uuid not null references auth.users(id) on delete cascade,
  quota_date date not null,
  reserved_count integer not null default 0 check (reserved_count >= 0),
  completed_count integer not null default 0 check (completed_count >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (user_id, quota_date)
);

create index if not exists daily_conversion_usage_quota_date_idx
  on app.daily_conversion_usage (quota_date);

alter table app.conversion_jobs
  add column if not exists quota_date date;

update app.conversion_jobs
set quota_date = timezone('Asia/Jakarta', created_at)::date
where quota_date is null;

alter table app.conversion_jobs
  add column if not exists quota_status text;

update app.conversion_jobs
set quota_status = case
  when status = 'done' then 'completed'
  when status in ('queued', 'processing') then 'reserved'
  else 'refunded'
end
where quota_status is null;

alter table app.conversion_jobs
  alter column quota_date set not null;

alter table app.conversion_jobs
  alter column quota_status set not null;

-- +goose StatementBegin
do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'conversion_jobs_quota_status_check'
  ) then
    alter table app.conversion_jobs
      add constraint conversion_jobs_quota_status_check
      check (quota_status in ('reserved', 'completed', 'refunded'));
  end if;
end $$;
-- +goose StatementEnd

insert into app.daily_conversion_usage (user_id, quota_date, reserved_count, completed_count)
select
  user_id,
  quota_date,
  sum(case when quota_status = 'reserved' then 1 else 0 end)::integer,
  sum(case when quota_status = 'completed' then 1 else 0 end)::integer
from app.conversion_jobs
group by user_id, quota_date
on conflict (user_id, quota_date) do update
set reserved_count = excluded.reserved_count,
    completed_count = excluded.completed_count;

drop trigger if exists touch_daily_conversion_usage_updated_at on app.daily_conversion_usage;
create trigger touch_daily_conversion_usage_updated_at
before update on app.daily_conversion_usage
for each row execute function app.touch_updated_at();

-- +goose Down
drop trigger if exists touch_daily_conversion_usage_updated_at on app.daily_conversion_usage;
drop table if exists app.daily_conversion_usage;
alter table app.conversion_jobs drop constraint if exists conversion_jobs_quota_status_check;
alter table app.conversion_jobs drop column if exists quota_status;
alter table app.conversion_jobs drop column if exists quota_date;
