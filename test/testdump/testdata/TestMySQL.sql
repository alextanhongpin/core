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
v1: 2023-07-10T22:13:11.941815+08:00
v2: 2023-07-10T22:13:11.941816+08:00


-- Normalized
SELECT *
  FROM `users`
  WHERE `name` LIKE ANY(:1)
    AND `age` > :age
    AND `created_at` IN (:v2)
    AND `deleted_at` IS NULL
    AND `description` = :description
    AND `email` = :email
    AND `is_active` = TRUE
    AND `last_logged_in_at` > :v1
    AND `subscription` IN ::2

-- Vars
"1": '{Foo,bar,%oo%}'
"2": freemium, premium
age: "13"
description: foo bar walks in a bar, h'a
email: john.doe@mail.com


-- Result
- id: 2286329016095019508
  name: Alice
- id: 4190232805252081789
  name: Bob
