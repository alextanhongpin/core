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
v1: "2023-07-11"
v2: "2023-07-11"


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
- id: 4382821114756197659
  name: Alice
- id: 3397901520974381966
  name: Bob
