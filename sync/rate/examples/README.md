# Rate examples

Illustrative integrations for [sync/rate](../README.md): HTTP request and error tracking, service health reports, adaptive admission, external API calls, and backend selection.

These files belong to the `github.com/alextanhongpin/core/sync/rate` module. From that module directory, inspect or compile the examples with:

```sh
go doc ./examples
go build ./examples
```

`RunExamples` demonstrates the helpers. Database and backend operations are simulated. `ExternalAPIClient.MakeRequest` currently leaves its returned response nil, and `GetCurrentLimit` reports the base limit. Treat these as demonstrations to adapt, rather than complete service implementations.
