-- Habilita funções de hash de senha (se ainda não existir)
create extension if not exists pgcrypto;

-- 1) Sequência
create sequence if not exists barrel.seq_users;

-- 2) Função para gerar código AAA9999 a partir do ID
create or replace function barrel.generate_user_code(user_id bigint)
returns varchar
language plpgsql
strict
immutable
as $$
declare
  x bigint;
  c1 int;
  c2 int;
  c3 int;
  letters text;
  numbers text;
begin
  -- 26^3 = 17576
  x := user_id % 17576;

  c1 := (x / 676);           -- 26^2
  c2 := (x % 676) / 26;      -- 26^1
  c3 := x % 26;              -- 26^0

  letters := chr(65 + c1) || chr(65 + c2) || chr(65 + c3);
  numbers := to_char((user_id % 10000), 'FM0000');

  return letters || numbers; -- AAA9999
end;
$$;

-- 3) Tabela já incluindo CODE
create table if not exists barrel.users
(
  id         bigint primary key default nextval('barrel.seq_users'),
  type       varchar(20)              not null check ( type in ('A','U') ),
  username   varchar(50) unique       not null,
  name       text                     not null,
  email      varchar(50) unique       not null,
  password   varchar(200)             not null,
  status     char(1)                  not null default 'A' check ( status in ('A','I','D') ),
  plan_id    bigint references barrel.subscription_plans(id),
  code       varchar(7)               not null,  -- AAA9999
  biometric_login      boolean                  not null default false,
  biometric_edit       boolean                  not null default false,
  biometric_remove     boolean                  not null default false,
  created_at timestamp with time zone not null default current_timestamp,
  updated_at timestamp with time zone not null default current_timestamp,
  deleted_at timestamp with time zone,
  constraint users_code_ck check (code ~ '^[A-Z]{3}[0-9]{4}$'),
  constraint users_code_uk unique (code)
);

comment on table barrel.users is 'Users of the platform';
comment on column barrel.users.type is 'A: Athlete, N: Nutritionist, P: Personal';
comment on column barrel.users.status is 'A: Active, I: Inactive, D: Deleted';
comment on column barrel.users.plan_id is 'FK to subscription_plans (user subscription plan)';
comment on column barrel.users.code is 'Share code AAA9999 generated from user id';
comment on column barrel.users.biometric_login  is 'Se true: exigir biometria para login';
comment on column barrel.users.biometric_edit   is 'Se true: exigir biometria para editar dispositivo';
comment on column barrel.users.biometric_remove is 'Se true: exigir biometria para remover/desconectar dispositivo';

-- 4) Trigger: hash de senha e geração do CODE no INSERT
create or replace function fnc_trg_users_biu() 
returns trigger as $$
begin
  -- atualiza updated_at sempre
  new.updated_at := current_timestamp;

  -- hash de senha no INSERT ou quando alterada
  if (TG_OP = 'INSERT' or coalesce(new.password, '') <> coalesce(old.password, '')) then
    new.password := crypt(new.password, gen_salt('bf'));
  end if;

  -- gera CODE somente no INSERT quando não informado
  if (TG_OP = 'INSERT') then
    if new.code is null or new.code = '' then
      -- NEW.id já vem do default nextval antes do BEFORE
      new.code := barrel.generate_user_code(new.id);
    end if;
  end if;

  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_users_biu on barrel.users;
create trigger trg_users_biu
  before insert or update on barrel.users
  for each row
  execute function fnc_trg_users_biu();
