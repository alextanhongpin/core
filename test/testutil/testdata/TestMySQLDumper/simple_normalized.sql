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
 "v1": "52b85874-a785-49bf-ad8b-07949d9cc785"
}


-- Rows
{
 "ID": 1,
 "Name": "Alice"
}