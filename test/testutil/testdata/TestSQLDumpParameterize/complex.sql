-- Query
SELECT
  *
FROM
  users
WHERE
  email = $2
  AND deleted_at IS NULL
  AND last_logged_in_at > $1
  AND description = $3
  AND subscription IN ($4, $5)
  AND age > $6
  AND is_active = $7
  AND name LIKE ANY ($8);


-- Args
{
 "$1": "2023-06-23",
 "$2": "john.doe@mail.com",
 "$3": "foo bar walks in a bar, h''a",
 "$4": "freemium",
 "$5": "premium",
 "$6": 13,
 "$7": true,
 "$8": "{Foo,bar,%oo%}"
}


-- Rows
null
