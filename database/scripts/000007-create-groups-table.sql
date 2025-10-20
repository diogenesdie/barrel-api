create sequence if not exists barrel.seq_groups;

create table if not exists barrel.groups
(
  id             bigint primary key default nextval('barrel.seq_groups'),
  user_id        bigint not null references barrel.users(id) on delete cascade,
  name           varchar(100)              not null,
  icon           varchar(100),
  is_default     boolean                  not null default false,
  is_share_group boolean                  not null default false,
  position       int                       not null default 0,
  created_at     timestamp with time zone  not null default current_timestamp,
  updated_at     timestamp with time zone  not null default current_timestamp
);

comment on table barrel.groups is 'Grupos de dispositivos criados pelo usuário';
comment on column barrel.groups.user_id is 'FK para users (dono do grupo)';
comment on column barrel.groups.position is 'Ordem de exibição do grupo';
comment on column barrel.groups.is_default is 'Indica se o grupo é o padrão (não pode ser deletado)';

create or replace function fnc_trg_groups_biu() 
returns trigger as $$
begin
  new.updated_at := current_timestamp;
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_groups_biu on barrel.groups;
create trigger trg_groups_biu
  before insert or update on barrel.groups
  for each row
  execute function fnc_trg_groups_biu();