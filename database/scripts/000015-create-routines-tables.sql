-- Routines: automated action sequences triggered by device state or schedule
create sequence if not exists barrel.seq_routines;

create table if not exists barrel.routines
(
    id             bigint primary key default nextval('barrel.seq_routines'),
    user_id        bigint                   not null references barrel.users (id),
    name           varchar(100)             not null,
    enabled        boolean                  not null default true,
    trigger_type   varchar(20)              not null check (trigger_type in ('device', 'schedule')),
    -- device trigger fields
    trigger_device_id    bigint references barrel.smart_devices (id),
    trigger_expected_state jsonb,
    -- schedule trigger fields
    trigger_cron   varchar(100),
    created_at     timestamp with time zone not null default current_timestamp,
    updated_at     timestamp with time zone not null default current_timestamp,
    deleted_at     timestamp with time zone
);

comment on table barrel.routines is 'User-defined routines triggered by device state changes or time schedules';
comment on column barrel.routines.trigger_cron is 'Standard 5-field cron expression (UTC)';

-- Routine actions: ordered list of steps executed when a routine fires
create sequence if not exists barrel.seq_routine_actions;

create table if not exists barrel.routine_actions
(
    id         bigint primary key default nextval('barrel.seq_routine_actions'),
    routine_id bigint                   not null references barrel.routines (id) on delete cascade,
    action_type varchar(20)             not null check (action_type in ('device', 'scene')),
    device_id  bigint references barrel.smart_devices (id),
    command    varchar(100),
    scene_id   bigint references barrel.scenes (id),
    sort_order int                      not null default 0,
    created_at timestamp with time zone not null default current_timestamp,
    constraint chk_routine_action_target check (
        (action_type = 'device' and device_id is not null and command is not null) or
        (action_type = 'scene'  and scene_id  is not null)
    )
);

comment on table barrel.routine_actions is 'Ordered actions (device command or scene activation) within a routine';

create index if not exists idx_routines_user_id on barrel.routines (user_id) where deleted_at is null;
create index if not exists idx_routines_trigger_device on barrel.routines (trigger_device_id) where deleted_at is null and enabled = true;
create index if not exists idx_routine_actions_routine_id on barrel.routine_actions (routine_id);
