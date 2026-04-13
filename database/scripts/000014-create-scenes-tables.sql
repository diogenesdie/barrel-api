-- Scenes: named collections of device actions
create sequence if not exists barrel.seq_scenes;

create table if not exists barrel.scenes
(
    id         bigint primary key default nextval('barrel.seq_scenes'),
    user_id    bigint                   not null references barrel.users (id),
    name       varchar(100)             not null,
    icon       varchar(50),
    created_at timestamp with time zone not null default current_timestamp,
    updated_at timestamp with time zone not null default current_timestamp,
    deleted_at timestamp with time zone
);

comment on table barrel.scenes is 'User-defined scenes that group multiple device actions';

-- Scene actions: ordered list of device commands within a scene
create sequence if not exists barrel.seq_scene_actions;

create table if not exists barrel.scene_actions
(
    id         bigint primary key default nextval('barrel.seq_scene_actions'),
    scene_id   bigint                   not null references barrel.scenes (id) on delete cascade,
    device_id  bigint                   not null references barrel.smart_devices (id),
    command    varchar(100)             not null,
    sort_order int                      not null default 0,
    created_at timestamp with time zone not null default current_timestamp
);

comment on table barrel.scene_actions is 'Ordered device commands that compose a scene';

create index if not exists idx_scene_actions_scene_id on barrel.scene_actions (scene_id);
create index if not exists idx_scenes_user_id on barrel.scenes (user_id) where deleted_at is null;
