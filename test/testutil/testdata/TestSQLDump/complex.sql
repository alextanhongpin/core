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
  email = 'john.doe@mail.com'
  AND deleted_at IS NULL
  AND last_logged_in_at > $1
  AND description = e'foo bar walks in a bar, h\'a'
  AND subscription IN ('freemium', 'premium')
  AND age > 13
  AND is_active = true
  AND name LIKE ANY ('{Foo,bar,%oo%}');


-- Args
{
 "$1": "2023-06-23"
}


-- Rows
null