-- Query
SELECT *
  FROM users
  WHERE id = $1


-- Args
$1: "1"


-- Normalized
SELECT *
  FROM users
  WHERE id = $1


-- Vars
$1: ""