-- Query
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL


-- Args
v1: a6935281-15ae-44a3-99c1-32d44ecb4f0f


-- Normalized
SELECT *
  FROM `users`
  WHERE `id` = :v1
    AND `deleted_at` IS NULL


-- Result
ID: 1
Name: Alice
