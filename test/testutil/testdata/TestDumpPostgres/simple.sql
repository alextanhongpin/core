-- Query
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL

-- Args
$1: 04bd405a-47f3-4fb5-bb67-3c267879fb87


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
