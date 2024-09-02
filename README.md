#  core


[![](https://godoc.org/github.com/alextanhongpin/core?status.svg)](http://godoc.org/github.com/alextanhongpin/core)

Useful collection of dependencies required to build microservices.


WIP:
- add suggested folder structure for APIs, domain layers etc.

## Project structure for Microservice

```mermaid
---
title: Go package structure
---
flowchart
    p0[User]


    subgraph b0[microservice]
        adapter
        presentation
        domain
        usecase
    end

    p0 --> presentation
    presentation --> usecase
    usecase --> adapter & domain
```


Other packages

- https://github.com/alextanhongpin/autocomplete
- https://github.com/alextanhongpin/dbtx
- https://github.com/alextanhongpin/errors
- https://github.com/alextanhongpin/money
- https://github.com/alextanhongpin/passwd
- https://github.com/alextanhongpin/passwordless
- https://github.com/alextanhongpin/profane
- https://github.com/alextanhongpin/stringcases
- https://github.com/alextanhongpin/stringdist
- ~https://github.com/alextanhongpin/builder~
- ~https://github.com/alextanhongpin/circuit~
- ~https://github.com/alextanhongpin/clash~
- ~https://github.com/alextanhongpin/constructor~
- ~https://github.com/alextanhongpin/dataloader2~
- ~https://github.com/alextanhongpin/dataloader3~
- ~https://github.com/alextanhongpin/dataloader~
- ~https://github.com/alextanhongpin/getter~
- ~https://github.com/alextanhongpin/goql~
- ~https://github.com/alextanhongpin/mapper~
- ~https://github.com/alextanhongpin/promise~
- ~https://github.com/alextanhongpin/set~
- ~https://github.com/alextanhongpin/transition~
- ~https://github.com/alextanhongpin/typeahead~
