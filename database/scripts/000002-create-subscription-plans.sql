create sequence barrel.seq_subscription_plans;

create table barrel.subscription_plans
(
  id                   bigint primary key default nextval('barrel.seq_subscription_plans'),
  name                 varchar(50)              not null,
  price                numeric(10,2),                  
  currency             varchar(10)              not null default 'BRL',
  max_devices          int,
  local_communication  boolean                  not null default true,
  mqtt_included        boolean                  not null default true,
  technical_support    boolean                  not null default false,
  priority_support     boolean                  not null default false,
  custom_widgets       boolean                  not null default false,
  custom_integrations  boolean                  not null default false,
  is_popular           boolean                  not null default false,
  is_custom_price      boolean                  not null default false,
  created_at           timestamp with time zone not null default current_timestamp,
  updated_at           timestamp with time zone not null default current_timestamp
);

comment on table barrel.subscription_plans is 'Available subscription plans for the platform';
comment on column barrel.subscription_plans.is_custom_price is 'Indicates if the plan requires custom quotation (Enterprise)';

insert into barrel.subscription_plans 
(name, price, currency, max_devices, local_communication, mqtt_included, technical_support, priority_support, custom_widgets, custom_integrations, is_popular, is_custom_price)
values
-- Plano Gratuito
('Free', 0, 'BRL', 3, true, true, false, false, false, false, false, false),
-- Plano Básico
('Basic', 29, 'BRL', 10, true, true, true, false, false, false, false, false),
-- Plano Intermediário (mais popular)
('Intermediate', 59, 'BRL', 30, true, true, true, true, true, false, true, false),
-- Plano Avançado
('Advanced', 99, 'BRL', 50, true, true, true, true, false, true, false, false),
-- Plano Empresarial (valor personalizado)
('Enterprise', null, 'BRL', null, true, true, true, true, false, true, false, true);
