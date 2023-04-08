# cmd.make

Download useful Makefiles that can be composed together for ease of development.

Usage:

```bash
$ go run github.com/alextanhongpin/go-core-microservice/cmd/make -name docker
```


Currently available Makefiles:

- `docker`: easily start and stop docker-compose

WIP:
- `atlas`: manage database migration declaratively and through versioning
- `test`: executes golang tests, view test coverage
- `database`: other command database operations
- `air`: hot-reload for golang application
