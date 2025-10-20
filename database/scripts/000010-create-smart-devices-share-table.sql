-- Sequence para smart_devices_share
create sequence if not exists barrel.seq_smart_devices_share;

-- Tabela de preferências do usuário sobre um dispositivo compartilhado
create table if not exists barrel.smart_devices_share
(
  id               bigint primary key default nextval('barrel.seq_smart_devices_share'),
  device_share_id  bigint not null references barrel.device_shares(id) on delete cascade,
  device_id        bigint not null references barrel.smart_devices(id) on delete cascade,
  user_id          bigint not null references barrel.users(id) on delete cascade,
  
  group_id         bigint references barrel.groups(id) on delete set null,
  is_favorite      boolean not null default false,
  name             varchar(100),
  icon             varchar(100),

  created_at       timestamp with time zone not null default current_timestamp,
  updated_at       timestamp with time zone not null default current_timestamp,
  deleted_at       timestamp with time zone,

  constraint uq_sds_device_user unique (device_id, user_id)
);

comment on table barrel.smart_devices_share is 'Customizações de dispositivos compartilhados pelo usuário que recebeu';
comment on column barrel.smart_devices_share.device_share_id is 'FK para device_shares (identifica o compartilhamento do dispositivo)';
comment on column barrel.smart_devices_share.device_id is 'FK para smart_devices (dispositivo compartilhado)';
comment on column barrel.smart_devices_share.user_id is 'Usuário que recebeu o compartilhamento';
comment on column barrel.smart_devices_share.group_id is 'Grupo definido pelo usuário que recebeu o compartilhamento';
comment on column barrel.smart_devices_share.is_favorite is 'Se o usuário marcou o dispositivo compartilhado como favorito';
comment on column barrel.smart_devices_share.name is 'Nome customizado pelo usuário que recebeu o compartilhamento';
comment on column barrel.smart_devices_share.icon is 'Ícone customizado pelo usuário que recebeu o compartilhamento';

-- Trigger de atualização do updated_at
create or replace function fnc_trg_smart_devices_share_biu() 
returns trigger as $$
begin
  new.updated_at := current_timestamp;
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_smart_devices_share_biu on barrel.smart_devices_share;
create trigger trg_smart_devices_share_biu
  before insert or update on barrel.smart_devices_share
  for each row
  execute function fnc_trg_smart_devices_share_biu();
