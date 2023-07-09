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
v1: 3df9910e-c7c4-4d84-b127-a662f968841d



-- Result
ID: 1
Name: Alice
