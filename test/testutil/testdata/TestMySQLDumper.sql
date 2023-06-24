-- Query
select * from users where id = :id and age = :v1 and `status` in ::bv1 and subscription in ::bv2 and created_at > :created_at order by age desc limit :v2


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