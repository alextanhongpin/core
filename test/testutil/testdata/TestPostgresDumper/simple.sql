-- Query
SELECT
  *
FROM
  users
WHERE
  id = $1 AND deleted_at IS NULL;


-- Query Normalized
SELECT
  *
FROM
  users
WHERE
  id = $1 AND deleted_at IS NULL;


-- Args
{
 "$1": "194893f7-ddf3-4ad0-88f1-bef01e1e2c1f"
}


-- Result
{
 "ID": 1,
 "Name": "Alice"
}