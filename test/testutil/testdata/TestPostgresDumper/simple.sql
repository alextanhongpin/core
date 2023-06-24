-- Query
SELECT
  *
FROM
  users
WHERE
  id = $1 AND deleted_at IS NULL;


-- Args
{
 "$1": "8928d824-eac1-475f-87a7-ffbb4fcda175"
}


-- Rows
{
 "ID": 1,
 "Name": "Alice"
}