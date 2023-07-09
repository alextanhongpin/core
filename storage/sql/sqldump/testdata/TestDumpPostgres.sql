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
{"$1":"2023-07-09","$2":"2023-07-09"}

-- Result
[{"ID":3965163071539938567,"Name":"Alice"},{"ID":8743618494151964555,"Name":"Bob"}]