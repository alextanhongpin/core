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
$1: 4e2d0885-2354-4728-abd5-61bbb6711390



-- Result
ID: 1
Name: Alice
