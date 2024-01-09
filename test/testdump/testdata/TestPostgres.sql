-- query --
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

-- args --
$1: 2024-01-09T21:49:34.801339+08:00
$2: 2024-01-09T21:49:34.801339+08:00


-- normalized --
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

-- vars --
$1: $1
$2: $2
$3: john.doe@mail.com
$4: foo bar walks in a bar, h''a
$5: freemium
$6: premium
$7: "13"
$8: "true"
$9: '{Foo,bar,%oo%}'
$10: "1"
$11: "2"


-- result --
- id: 1477336475297101788
  name: Alice
- id: 7829061172683113698
  name: Bob
