-- Botões de controle remoto IR/RF sincronizados pelo app Flutter

create sequence if not exists barrel.seq_device_buttons;

create table if not exists barrel.device_buttons
(
  id            bigint primary key default nextval('barrel.seq_device_buttons'),
  device_id     bigint       not null references barrel.smart_devices(id) on delete cascade,
  original_name varchar(100) not null,
  protocol      varchar(50)  not null default '',
  address       integer      not null default 0,
  command       integer      not null default 0,
  label         varchar(100) not null,
  created_at    timestamp with time zone not null default current_timestamp,
  updated_at    timestamp with time zone not null default current_timestamp,
  unique (device_id, original_name)
);

comment on table barrel.device_buttons is 'Botões IR/RF dos dispositivos, sincronizados pelo app';
comment on column barrel.device_buttons.original_name is 'Identificador fixo do firmware (ex: BTN_1), chave estável para comandos MQTT';
comment on column barrel.device_buttons.label is 'Nome editável pelo usuário, usado como friendly name na Alexa';

create or replace function fnc_trg_device_buttons_biu()
returns trigger as $$
begin
  new.updated_at := current_timestamp;
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_device_buttons_biu on barrel.device_buttons;
create trigger trg_device_buttons_biu
  before insert or update on barrel.device_buttons
  for each row
  execute function fnc_trg_device_buttons_biu();
