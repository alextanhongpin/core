# cmd.make

Download useful Makefiles that can be composed together for ease of development.

Usage:

```bash
$ go run github.com/alextanhongpin/core/cmd/make -name docker
```


Currently available Makefiles:

- `docker`: easily start and stop docker-compose
- `atlas`: manage database migration declaratively and through versioning

WIP:
- `test`: executes golang tests, view test coverage
- `database`: other command database operations
- `air`: hot-reload for golang application


## Combining Makefiles


You can create a root `Makefile` and include the other makefiles:

```Makefile
include .env
export


include Makefile.*.mk
```
