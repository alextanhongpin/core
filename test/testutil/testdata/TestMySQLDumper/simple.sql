-- Query
SELECT *
  FROM users
  WHERE id = 1
    AND age = :v1
    AND `status` IN ('pending',
                     'success')
    AND subscription IN ('gold',
                         'silver')
    AND created_at > '2023-01-01'
  ORDER BY age DESC
  LIMIT :v2


-- Args
{
 "v1": 13,
 "v2": 10
}


-- Rows
null