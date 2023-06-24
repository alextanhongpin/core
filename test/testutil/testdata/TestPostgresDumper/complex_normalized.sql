-- Query
SELECT
  *
FROM
  users
WHERE
  email = 'john.doe@mail.com'
  AND deleted_at IS NULL
  AND last_logged_in_at > $1
  AND description = e'foo bar walks in a bar, h\'a'
  AND subscription IN ('freemium', 'premium')
  AND age > 13
  AND is_active = true
  AND name LIKE ANY ('{Foo,bar,%oo%}');


-- Query Normalized
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
 "$1": "2023-06-24",
 "$2": "john.doe@mail.com",
 "$3": "foo bar walks in a bar, h''a",
 "$4": "freemium",
 "$5": "premium",
 "$6": 13,
 "$7": true,
 "$8": "{Foo,bar,%oo%}"
}


-- Rows
[
 {
  "ID": 6351818813913898639,
  "Name": "Alice"
 },
 {
  "ID": 3903176830726961892,
  "Name": "Bob"
 }
]