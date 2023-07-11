-- Query
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL

-- Args
v1: 7115bcb4-b765-43fe-9770-bb1ce48cad10


-- Normalized
SELECT *
  FROM `users`
  WHERE `id` = :v1
    AND `deleted_at` IS NULL

-- Vars
{}


-- Result
ID: 1
Name: Alice
