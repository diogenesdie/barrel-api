-- Códigos de autorização OAuth2 para account linking com Alexa

create sequence if not exists barrel.seq_oauth_codes;

create table if not exists barrel.oauth_codes
(
  id           bigint primary key default nextval('barrel.seq_oauth_codes'),
  code         varchar(128)             not null unique,
  user_id      bigint                   not null references barrel.users(id) on delete cascade,
  redirect_uri text                     not null,
  expires_at   timestamp with time zone not null,
  used         boolean                  not null default false,
  created_at   timestamp with time zone not null default current_timestamp
);

create index if not exists idx_oauth_codes_code on barrel.oauth_codes(code);

comment on table barrel.oauth_codes is 'Códigos de autorização OAuth2 para account linking (Alexa, etc.)';
comment on column barrel.oauth_codes.code is 'Código aleatório de 64 bytes em hex (128 chars), uso único';
comment on column barrel.oauth_codes.used is 'Marcado como true após consumo; impede replay attacks';
