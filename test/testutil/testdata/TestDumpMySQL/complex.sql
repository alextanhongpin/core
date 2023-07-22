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
v1: "2023-07-22"


-- Normalized
SELECT *
  FROM `users`
  WHERE `email` = :email
    AND `deleted_at` IS NULL
    AND `last_logged_in_at` > :v1
    AND `created_at` IN (:v2)
    AND `description` = :description
    AND `subscription` IN ::1
    AND `age` > :age
    AND `is_active` = TRUE
    AND `name` LIKE ANY(:2)


-- Vars
"1": freemium, premium
"2": '{Foo,bar,%oo%}'
age: "13"
description: foo bar walks in a bar, h'a
email: john.doe@mail.com


-- Result
- id: 8511647005632270790
  name: Alice
- id: 1455461433628230928
  name: Bob
