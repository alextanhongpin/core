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


-- Query Normalized
SELECT *
  FROM users
  WHERE email = $3
    AND deleted_at IS NULL
    AND last_logged_in_at > $1
    AND created_at IN ($2)
    AND description = $4
    AND subscription IN ($5,
                         $6)
    AND age > $7
    AND is_active = $8
    AND name LIKE ANY($9)
    AND id <> ALL(ARRAY[$10, $11])


-- Args
$1: "2023-07-09"
$3: john.doe@mail.com
$4: foo bar walks in a bar, h''a
$5: freemium
$6: premium
$7: 13
$8: true
$9: '{Foo,bar,%oo%}'
$10: 1
$11: 2



-- Result
- id: 8742928967736104934
  name: Alice
- id: 5887499256977334832
  name: Bob
