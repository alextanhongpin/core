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
v1: 2024-01-09T02:38:23.382451+08:00
v2: 2024-01-09T02:38:23.382452+08:00


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
- id: 7012117209322283376
  name: Alice
- id: 4528519728315617392
  name: Bob