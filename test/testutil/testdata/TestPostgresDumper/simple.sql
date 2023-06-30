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
{
 "$1": "7413e9bf-75bc-4383-9a51-6a0caece1a08"
}


-- Result
{
 "ID": 1,
 "Name": "Alice"
}