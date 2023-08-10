# Retry 

Package retry allows handler to be retried using linear or exponential backoffs.

The package exposes two methods, `Query` and `Exec`.

If the handler does not return a value, then use `Exec`. `Query` allows a generic value to be returned, reducing the need for manually casting the types.
