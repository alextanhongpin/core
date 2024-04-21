# dsync

Package `dsync` stands for *distributed sync*, and contains a collection of packages that is useful for managing concurrent read/writes across different services.

Most packages are built based on the concept of distributed lock, acquiring exclusive locks to a given resource, typically to perform some operation such as idempotent creation, or atomic state transition.

There are several options, such as

- redis lock
- etcd
- zookeeper
- dynamodb
- s3 or minio lock


The issue with distributed implementation is that it requires a distributed store.
In case of failures, we can just fallback to the in-memory implementation that has the same interface.
