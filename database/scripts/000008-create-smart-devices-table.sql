create sequence if not exists barrel.seq_smart_devices;

create table if not exists barrel.smart_devices
(
  id                    bigint primary key default nextval('barrel.seq_smart_devices'),
  device_id             varchar(100)               not null,
  user_id               bigint not null references barrel.users(id) on delete cascade,
  group_id              bigint references barrel.groups(id) on delete set null,
  name                  varchar(100)              not null,
  type                  varchar(50)               not null,
  icon                  varchar(100),
  ip                    inet,
  iv_key                varchar(200),
  state                 varchar(10)               not null default 'off' check (state in ('on','off')),
  is_favorite           boolean                   not null default false,
  ssid                  varchar(100),
  communication_mode    varchar(10)               not null default 'auto' check (communication_mode in ('auto','local')),
  created_at            timestamp with time zone not null default current_timestamp,
  updated_at            timestamp with time zone not null default current_timestamp,
  deleted_at            timestamp with time zone
);

comment on table barrel.smart_devices is 'Dispositivos inteligentes cadastrados pelo usuário';
comment on column barrel.smart_devices.user_id is 'FK para users (dono do dispositivo)';
comment on column barrel.smart_devices.group_id is 'FK para groups (grupo ao qual pertence o dispositivo)';
comment on column barrel.smart_devices.iv_key is 'Chave IV usada para criptografia de comunicação';
comment on column barrel.smart_devices.state is 'Estado atual do dispositivo: on/off';
comment on column barrel.smart_devices.is_favorite is 'Indica se o dispositivo está marcado como favorito';
comment on column barrel.smart_devices.communication_mode is 'Modo de comunicação: auto (MQTT) ou local (HTTP)';

create or replace function fnc_trg_smart_devices_biu() 
returns trigger as $$
begin
  new.updated_at := current_timestamp;
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_smart_devices_biu on barrel.smart_devices;
create trigger trg_smart_devices_biu
  before insert or update on barrel.smart_devices
  for each row
  execute function fnc_trg_smart_devices_biu();