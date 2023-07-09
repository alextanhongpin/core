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
{":v1":"2023-07-09",":v2":"2023-07-09"}

-- Result
[{"ID":6158519075357417005,"Name":"Alice"},{"ID":374432378813503970,"Name":"Bob"}]