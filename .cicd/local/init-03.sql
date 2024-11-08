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


CREATE TABLE sales_olap_42
(
    region_id        INT  NOT NULL,
    month            DATE NOT NULL,
    product_category VARCHAR(50),
    total_sales      NUMERIC(12, 2),
    total_quantity   INT,
    avg_sale_price   NUMERIC(8, 2),
    PRIMARY KEY (region_id, month, product_category)
);



INSERT INTO sales_olap_42 (region_id, month, product_category, total_sales, total_quantity, avg_sale_price)
SELECT region_id,
       month::date,
       category                                    AS product_category,
       ROUND((RANDOM() * 4000 + 1000)::numeric, 2) AS total_sales,
       quantity                                    AS total_quantity,
       ROUND((total_sales / quantity)::numeric, 2) AS avg_sale_price
FROM (SELECT region_id,
             category,
             (date '2024-01-01' + (gs * '1 month'::interval))::date AS month,
             FLOOR(RANDOM() * 90 + 10)                              AS quantity
      FROM generate_series(1, 1200) AS gs,                 -- Generates data for 12 months
           (VALUES (1), (2), (3)) AS regions(region_id), -- Region IDs
           (VALUES ('Electronics'), ('Furniture'), ('Clothing')) AS categories(category) -- Product categories
     ) AS base_data
         CROSS JOIN LATERAL (
    SELECT ROUND((RANDOM() * 4000 + 1000)::numeric, 2) AS total_sales
    ) AS random_sales;
