-- Query
SELECT *
  FROM users
  WHERE email = 'john.doe@mail.com'
    AND deleted_at IS NULL
    AND last_logged_in_at > :v1
    AND description = 'foo bar walks in a bar, h\'a'
    AND subscription IN ('freemium',
                         'premium')
    AND age > 13
    AND is_active = TRUE
    AND `name` like ANY('{Foo,bar,%oo%}')


-- Args
{
 "v1": "2023-06-25"
}


-- Rows
[
 {
  "ID": 1033728696049817658,
  "Name": "Alice"
 },
 {
  "ID": 808767495125382765,
  "Name": "Bob"
 }
]