-- Query
SELECT *
  FROM users
  WHERE id = :v1


-- Args
v1: "1"


-- Normalized
SELECT *
  FROM `users`
  WHERE `id` = :v1