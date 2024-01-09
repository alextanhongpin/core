-- query --
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL

-- args --
$1: ffc3c5de-203f-435f-bef3-091a00c8612e


-- normalized --
SELECT *
  FROM users
  WHERE id = $1
    AND deleted_at IS NULL

-- vars --
$1: $1


-- result --
ID: 1
Name: Alice
