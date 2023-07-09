-- Query
SELECT *
  FROM users
  WHERE email = 'john.doe@mail.com'
    AND deleted_at IS NULL
    AND last_logged_in_at > $1
    AND created_at IN ($2)
    AND description = 'foo bar walks in a bar, h''a'
    AND subscription IN ('freemium',
                         'premium')
    AND age > 13
    AND is_active = TRUE
    AND name LIKE ANY('{Foo,bar,%oo%}')
    AND id <> ALL(ARRAY[1, 2])

-- Args
$1: 2023-07-10T02:18:48.972295+08:00
$2: 2023-07-10T02:18:48.972296+08:00


-- Result
- id: 5002517457082324061
  name: Alice
- id: 383751125311202439
  name: Bob
