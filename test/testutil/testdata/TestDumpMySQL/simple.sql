-- Query
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL

-- Args
:v1: a1029881-1beb-4687-ae01-25def15a8df2


-- Normalized
SELECT *
  FROM `users`
  WHERE `deleted_at` IS NULL
    AND `id` = :v1

-- Vars
{}


-- Result
ID: 1
Name: Alice
