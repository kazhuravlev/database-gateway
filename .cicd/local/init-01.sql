create table clients
(
    id      text primary key,
    name    text,
    balance int,
    email   text
);


insert into clients(id, name, balance, email)
values ('1', 'kirill', 444, 'k@example.com'),
       ('2', 'andrew', 333, 'a@example.com');