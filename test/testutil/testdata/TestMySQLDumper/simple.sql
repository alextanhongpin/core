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
v1: 0a521312-97ad-4b2c-a00d-38f927459f71



-- Result
ID: 1
Name: Alice
