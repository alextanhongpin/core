-- Query
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL


-- Query Normalized
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL


-- Args
$1: a6082ac5-e55e-4004-a852-49a5a682f1d9



-- Result
ID: 1
Name: Alice
