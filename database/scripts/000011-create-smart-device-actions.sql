create sequence if not exists barrel.seq_smart_device_actions;

create table if not exists barrel.smart_device_actions
(
  id               bigint primary key default nextval('barrel.seq_smart_device_actions'),

  trigger_device_id bigint not null
    references barrel.smart_devices(id)
    on delete cascade,

  trigger_event     varchar(20) not null
    check (trigger_event in ('single_click','double_click','triple_click','long_click')),

  target_device_id  bigint not null
    references barrel.smart_devices(id)
    on delete cascade,

  action_type       varchar(20) not null
    check (action_type in ('on','off','toggle','pulse','release')),

  created_at        timestamp with time zone not null default current_timestamp,
  updated_at        timestamp with time zone not null default current_timestamp
);

comment on table barrel.smart_device_actions is 'Mapeia ações executadas por um dispositivo do tipo Trigger sobre outros dispositivos.';
comment on column barrel.smart_device_actions.trigger_device_id is 'Dispositivo do tipo Trigger que dispara a ação.';
comment on column barrel.smart_device_actions.trigger_event is 'Evento que dispara a ação (single_click, double_click, triple_click, long_click).';
comment on column barrel.smart_device_actions.target_device_id is 'Dispositivo alvo que receberá a ação.';
comment on column barrel.smart_device_actions.action_type is 'Tipo de ação executada no dispositivo alvo (on/off/toggle/pulse/release).';

-- Atualiza o updated_at automaticamente
create or replace function fnc_trg_smart_device_actions_biu() 
returns trigger as $$
begin
  new.updated_at := current_timestamp;
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_smart_device_actions_biu on barrel.smart_device_actions;
create trigger trg_smart_device_actions_biu
  before insert or update on barrel.smart_device_actions
  for each row
  execute function fnc_trg_smart_device_actions_biu();
