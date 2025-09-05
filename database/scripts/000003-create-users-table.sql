create sequence if not exists barrel.seq_users;

create table if not exists barrel.users
(
  id         bigint primary key default nextval('barrel.seq_users'),
  type       varchar(20)              not null check ( type in ('A','N','P') ),
  username   varchar(50) unique       not null,
  name       text                     not null,
  email      varchar(50) unique       not null,
  password   varchar(200)             not null,
  status     char(1)                  not null default 'A' check ( status in ('A','I','D') ),
  plan_id    bigint references barrel.subscription_plans(id),
  created_at timestamp with time zone not null default current_timestamp,
  updated_at timestamp with time zone not null default current_timestamp,
  deleted_at timestamp with time zone
);

comment on table barrel.users is 'Users of the platform';
comment on column barrel.users.type is 'A: Athlete, N: Nutritionist, P: Personal';
comment on column barrel.users.status is 'A: Active, I: Inactive, D: Deleted';
comment on column barrel.users.plan_id is 'FK to subscription_plans (user subscription plan)';

create or replace function fnc_trg_users_biu() 
returns trigger as $$
begin
  --
  new.updated_at := current_timestamp;
  --
  if (TG_OP = 'INSERT' or coalesce(new.password, '') <> coalesce(old.password, '')) then
    --
    new.password := crypt(new.password, gen_salt('bf'));
    --
  end if;
  --
  return new;
end;
$$ language plpgsql;

drop trigger if exists trg_users_biu on barrel.users;
create trigger trg_users_biu
  before insert or update on barrel.users
  for each row
  execute function fnc_trg_users_biu();

insert into barrel.users (type, username, name, email, password, plan_id)
values ('A', 'teste', 'Usuário Teste', 'teste@teste.com', '1234', 1);
