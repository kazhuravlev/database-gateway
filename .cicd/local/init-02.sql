create table clients
(
    id    text primary key,
    name  text,
    email text
);


insert into clients(id, name, email)
values ('1', 'kirill', 'k@email.com'),
       ('2', 'andrew', 'a@email.com');