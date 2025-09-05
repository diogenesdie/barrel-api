create sequence barrel.seq_user_devices;

create table barrel.user_devices
(
  id         bigint primary key default nextval('barrel.seq_user_devices'),
  user_id    bigint references barrel.users(id)   not null,
  device_id  bigint references barrel.devices(id) not null,
  created_at timestamp with time zone             not null default current_timestamp,
  updated_at timestamp with time zone             not null default current_timestamp,
  deleted_at timestamp with time zone
);