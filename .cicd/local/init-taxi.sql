-- Database Gateway provides access to servers with ACL for safe and restricted database interactions.
-- Copyright (C) 2024  Kirill Zhuravlev
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU General Public License as published by
-- the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU General Public License for more details.
--
-- You should have received a copy of the GNU General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.

create table clients
(
    id                text primary key,
    full_name         text not null,
    phone             text not null unique,
    email             text not null unique,
    home_city         text not null,
    rating            numeric(3, 2) not null,
    joined_at         timestamp not null
);

create table cars
(
    id                text primary key,
    plate_number      text not null unique,
    model             text not null,
    color             text not null,
    year_made         int not null,
    service_status    text not null,
    current_city      text not null
);

create table drivers
(
    id                text primary key,
    full_name         text not null,
    phone             text not null unique,
    rating            numeric(3, 2) not null,
    status            text not null,
    city              text not null,
    car_id            text not null references cars(id)
);

create table transfers
(
    id                text primary key,
    client_id         text not null references clients(id),
    driver_id         text not null references drivers(id),
    car_id            text not null references cars(id),
    pickup_address    text not null,
    dropoff_address   text not null,
    requested_at      timestamp not null,
    completed_at      timestamp,
    distance_km       numeric(6, 2) not null,
    duration_minutes  int not null,
    status            text not null,
    fare_amount_cents int not null
);

create table transactions
(
    id                 text primary key,
    transfer_id        text not null references transfers(id),
    payment_method     text not null,
    amount_cents       int not null,
    currency           text not null,
    status             text not null,
    processed_at       timestamp not null,
    external_reference text not null unique
);

select setseed(0.42);

insert into clients(id, full_name, phone, email, home_city, rating, joined_at)
select ('cli-' || gs)::text,
       format('Client %s', gs),
       format('+34-611-%04s', gs),
       format('client%02s@example.com', gs),
       case when gs % 3 = 0 then 'Barcelona'
            when gs % 3 = 1 then 'Madrid'
            else 'Valencia'
       end,
       round((4.10 + (random() * 0.89))::numeric, 2),
       timestamp '2025-01-01 09:00:00' + (gs * interval '18 hours')
from generate_series(100, 500) as gs;

insert into cars(id, plate_number, model, color, year_made, service_status, current_city)
select ('car-' || gs)::text,
       (
           case when gs % 3 = 0 then 'BCN'
                when gs % 3 = 1 then 'MAD'
                else 'VLC'
           end ||
           '-' ||
           4000 + gs
       ),
       (array['Toyota Prius', 'Tesla Model 3', 'Kia Niro', 'Hyundai Ioniq', 'Skoda Octavia', 'BYD Seal'])[1 + (gs % 6)],
       (array['white', 'black', 'silver', 'blue', 'red', 'gray'])[1 + (gs % 6)],
       2018 + (gs % 7),
       case when gs % 10 = 0 then 'maintenance'
            when gs % 7 = 0 then 'inactive'
            else 'active'
       end,
       case when gs % 3 = 0 then 'Barcelona'
            when gs % 3 = 1 then 'Madrid'
            else 'Valencia'
       end
from generate_series(100, 200) as gs;

insert into drivers(id, full_name, phone, rating, status, city, car_id)
select ('drv-' ||  gs)::text,
       format('Driver %s', gs),
       ('+34-622-' || gs),
       round((4.20 + (random() * 0.79))::numeric, 2),
       case when gs % 9 = 0 then 'offline'
            when gs % 5 = 0 then 'on_trip'
            else 'online'
       end,
       case when gs % 3 = 0 then 'Barcelona'
            when gs % 3 = 1 then 'Madrid'
            else 'Valencia'
       end,
       ('car-' || gs)
from generate_series(100, 200) as gs;

insert into transfers(
    id,
    client_id,
    driver_id,
    car_id,
    pickup_address,
    dropoff_address,
    requested_at,
    completed_at,
    distance_km,
    duration_minutes,
    status,
    fare_amount_cents
)
select ('trf-' || gs)::text,
       ('cli-' || 100 + (gs % 400)),
       ('drv-' || 100 + (gs % 100)),
       ('car-' || 100 + (gs % 100)),
       format('%s %s, %s',
           (array['Gran Via', 'Atocha', 'Diagonal', 'Aragon', 'Colon', 'Serrano'])[1 + (gs % 6)],
           10 + (gs % 90),
           case when gs % 3 = 0 then 'Barcelona'
                when gs % 3 = 1 then 'Madrid'
                else 'Valencia'
           end
       ),
       format('%s %s, %s',
           (array['Airport', 'Central Station', 'Old Town', 'Business District', 'Port', 'University'])[1 + (gs % 6)],
           1 + (gs % 20),
           case when gs % 3 = 0 then 'Barcelona'
                when gs % 3 = 1 then 'Madrid'
                else 'Valencia'
           end
       ),
       timestamp '2026-03-01 06:00:00' + (gs * interval '17 minutes'),
       case when gs % 8 = 0 then null
            else timestamp '2026-03-01 06:00:00' + (gs * interval '17 minutes') + ((12 + (gs % 35)) * interval '1 minute')
       end,
       round((2.50 + ((gs % 27) * 0.85) + random())::numeric, 2),
       12 + (gs % 35),
       case when gs % 15 = 0 then 'cancelled'
            when gs % 8 = 0 then 'in_progress'
            else 'completed'
       end,
       650 + (gs * 37)
from generate_series(1000, 5000) as gs;

insert into transactions(
    id,
    transfer_id,
    payment_method,
    amount_cents,
    currency,
    status,
    processed_at,
    external_reference
)
select ('txn-' || gs)::text,
       ('trf-' || 1000+(gs%4000)),
       (array['card', 'wallet', 'business_account', 'apple_pay'])[1 + (gs % 4)],
       650 + (gs * 37),
       'EUR',
       case when gs % 15 = 0 then 'voided'
            when gs % 8 = 0 then 'authorized'
            else 'captured'
       end,
       timestamp '2026-03-01 06:05:00' + (gs * interval '17 minutes'),
       format('pay_%s', substr(md5(('transfer-' || gs)), 1, 12))
from generate_series(1000, 9000) as gs;
