-- query --
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL

-- args --
v1: 7931fc7a-5ab0-4d9c-98e5-a8a12448eaa4


-- normalized --
SELECT *
  FROM `users`
  WHERE `id` = :v1
    AND `deleted_at` IS NULL

-- result --
ID: 1
Name: Alice
