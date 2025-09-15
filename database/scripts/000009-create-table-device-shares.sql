create sequence if not exists barrel.seq_device_shares;

create table if not exists barrel.device_shares
(
  id                 bigint primary key default nextval('barrel.seq_device_shares'),
  owner_id           bigint not null references barrel.users(id) on delete cascade,
  shared_with_id     bigint not null references barrel.users(id) on delete cascade,
  device_id           bigint references barrel.smart_devices(id) on delete cascade,
  group_id             bigint references barrel.groups(id) on delete cascade,
  status                 char(1) not null default 'P' check (status in ('P','A','R')),
  accepted_at               timestamp with time zone,
  revoked_at                  timestamp with time zone,
  created_at                     timestamp with time zone not null default current_timestamp,
  updated_at                     timestamp with time zone not null default current_timestamp,
  deleted_at                      timestamp with time zone,
  constraint chk_device_or_group check (
    (device_id is not null and group_id is null) or 
    (device_id is null and group_id is not null)
  )
);

comment on table barrel.device_shares is 'Compartilhamento de dispositivos ou grupos entre usuários';
comment on column barrel.device_shares.owner_id is 'Usuário dono que compartilha o recurso';
comment on column barrel.device_shares.shared_with_id is 'Usuário que recebe o compartilhamento';
comment on column barrel.device_shares.device_id is 'FK para smart_devices (quando compartilhar 1 dispositivo)';
comment on column barrel.device_shares.group_id is 'FK para groups (quando compartilhar um grupo inteiro)';
comment on column barrel.device_shares.status is 'P: Pending, A: Active, R: Revoked';
comment on column barrel.device_shares.accepted_at is 'Data/hora em que o usuário aceitou o compartilhamento';
comment on column barrel.device_shares.revoked_at is 'Data/hora em que o dono revogou o compartilhamento';

create or replace function fnc_trg_device_shares_biu() 
returns trigger as $$
begin
  new.updated_at := current_timestamp;

  if new.status = 'A' and (old.status is distinct from new.status) then
    new.accepted_at := current_timestamp;
    new.revoked_at := null;
  end if;

  if new.status = 'R' and (old.status is distinct from new.status) then
    new.revoked_at := current_timestamp;
  end if;

  if new.status = 'P' then
    new.accepted_at := null;
    new.revoked_at := null;
  end if;

  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_device_shares_biu on barrel.device_shares;
create trigger trg_device_shares_biu
  before insert or update on barrel.device_shares
  for each row
  execute function fnc_trg_device_shares_biu();
