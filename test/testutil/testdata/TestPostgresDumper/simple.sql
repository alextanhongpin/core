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
$1: 4668576a-bc0b-4765-bb89-a82a60c035b9



-- Result
ID: 1
Name: Alice
