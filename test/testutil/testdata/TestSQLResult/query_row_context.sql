-- query --
SELECT *
  FROM users
  WHERE id = $1

-- args --
$1: "1"


-- normalized --
SELECT *
  FROM users
  WHERE id = $1

-- vars --
$1: ""


-- result --
ID: "1"
Name: Alice
