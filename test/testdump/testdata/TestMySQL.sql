-- Query
SELECT *
  FROM users
  WHERE email = 'john.doe@mail.com'
    AND deleted_at IS NULL
    AND last_logged_in_at > :v1
    AND created_at IN (:v2)
    AND description = 'foo bar walks in a bar, h\'a'
    AND subscription IN ('freemium',
                         'premium')
    AND age > 13
    AND is_active = TRUE
    AND `name` like ANY('{Foo,bar,%oo%}')

-- Args
:v1: 2023-07-10T02:18:48.645579+08:00
:v2: 2023-07-10T02:18:48.645579+08:00


-- Result
- id: 7985689801699895449
  name: Alice
- id: 4923639882360704835
  name: Bob
