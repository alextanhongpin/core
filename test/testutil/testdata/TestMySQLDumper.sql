-- Query
SELECT *
  FROM users
  WHERE id = :id
    AND age = :v1
    AND `status` IN ::bv1
    AND subscription IN ::bv2
    AND created_at > :created_at
  ORDER BY age DESC
  LIMIT :v2


-- Args
{
 "bv1": [
  "pending",
  "success"
 ],
 "bv2": [
  "gold",
  "silver"
 ],
 "created_at": "2023-01-01",
 "id": "1",
 "v1": 13,
 "v2": 10
}


-- Rows
null