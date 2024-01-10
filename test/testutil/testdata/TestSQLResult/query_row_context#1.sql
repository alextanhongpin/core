-- query --
SELECT *
  FROM users
  WHERE id = $1

-- args --
$1: "2"


-- normalized --
SELECT *
  FROM users
  WHERE id = $1

-- vars --
$1: ""


-- result --
ID: "2"
Name: Bob
