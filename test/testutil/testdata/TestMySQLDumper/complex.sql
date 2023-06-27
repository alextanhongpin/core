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


-- Query Normalized
SELECT *
  FROM users
  WHERE email = :email
    AND deleted_at IS NULL
    AND last_logged_in_at > :v1
    AND created_at IN (:v2)
    AND description = :description
    AND subscription IN ::bv1
    AND age > :age
    AND is_active = TRUE
    AND `name` like ANY(:bv2)


-- Args
{
 "age": "13",
 "bv1": [
  "freemium",
  "premium"
 ],
 "bv2": "{Foo,bar,%oo%}",
 "description": "foo bar walks in a bar, h'a",
 "email": "john.doe@mail.com",
 "v1": "2023-06-27",
 "v2": null
}


-- Result
[
 {
  "ID": 9197004928671496586,
  "Name": "Alice"
 },
 {
  "ID": 2820128565906968551,
  "Name": "Bob"
 }
]