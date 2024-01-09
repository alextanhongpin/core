-- query --
SELECT *
  FROM users
  WHERE id = :v1

-- args --
v1: "1"


-- normalized --
SELECT *
  FROM `users`
  WHERE `id` = :v1

