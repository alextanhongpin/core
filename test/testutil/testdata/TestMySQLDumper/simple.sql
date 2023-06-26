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
{
 "v1": "e4741d26-9b95-49f6-8423-9cb015ad94db"
}


-- Result
{
 "ID": 1,
 "Name": "Alice"
}