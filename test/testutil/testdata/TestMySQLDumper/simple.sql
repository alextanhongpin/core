-- Query
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL


-- Query Normalized
SELECT *
  FROM users
  WHERE id = :v1
    AND deleted_at IS NULL


-- Args
v1: 28dc2951-5907-4791-81a9-bf1dbb93394a



-- Result
ID: 1
Name: Alice
