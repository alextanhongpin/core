-- Query
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL


-- Args
$1: 9ea5f063-7902-42ae-974e-ad59f39b2936


-- Normalized
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL


-- Vars
$1: $1


-- Result
ID: 1
Name: Alice
