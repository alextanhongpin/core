-- Query
SELECT
  *
FROM
  users
WHERE
  name = $1 AND age = $2;


-- Query Normalized
SELECT
  *
FROM
  users
WHERE
  name = $1 AND age = $2;


-- Args
{
 "$1": "John",
 "$2": 13
}


-- Rows
[
 {
  "ID": 1,
  "Name": "John"
 }
]