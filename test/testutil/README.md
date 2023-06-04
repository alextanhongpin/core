# Snapshot Testing

Snapshot testing is a type of software testing that compares the output of a software application to a known good state, or snapshot. This is done by taking a snapshot of the application's output at a specific point in time, and then comparing it to the snapshot at a later point in time. If the two snapshots are not identical, then the application has changed in an unexpected way, and a test failure is reported.

Snapshot testing is a valuable tool for ensuring that software applications do not change in an unexpected way. It can be used to detect changes in the application's output, UI, behavior, and data. Snapshot testing is often used in conjunction with other types of software testing, such as unit testing and integration testing.



## HTTP Dump

There is a standard library in golang to dump the HTTP request and response. However, the json output response is not pretty-printed.

```go
package snapshottest

import (
	"net/http"
	"net/http/httputil"
	"strings"
)

func dump(w *http.Response, r *http.Request) (string, error) {
	req, err := httputil.DumpRequest(r, true)
	if err != nil {
		return "", err
	}

	res, err := httputil.DumpResponse(w, true)
	if err != nil {
		return "", err
	}

	output := make([]string, 3)
	output[0] = string(req)
	output[2] = string(res)

	return strings.Join(output, "\n"), nil
}
```

## Reference

https://solidstudio.io/blog/snapshot-testing
