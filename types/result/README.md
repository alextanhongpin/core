# Result

 There are reasons to define result as interface, mainly to fulfil the return
 type of certain function/interface, yet keeping the original type.
 In short, we want to return multiple types, but keeping the return args to
 only two.
 For example, a rate-limit implementation may have the following return type:


 ```go
 function RateLimit(key string) (allow bool, err error) {
	 /* !REDACTED */
 }
 ```

 What if there are more information required, such as the rate limit
 remaining, reset and limit?
 Instead of returning `allow`, we can probably return a struct:

```go
type RateLimitResult struct {
 	Limit int
  Remaining int
  Reset time.Duration
}

function RateLimit(key string) (*RateLimitResult, error)
  if rateLimited {
 	 return &RateLimitResult{}, ErrRateLimited
  }

 return
}
```

But now we have another issue, which is we now expect the returned result to be non-empty when the error is not nil.

This can lead to poor API design. We can choose to return error only for infrastructure-related errors, and ensure a null object is always returned for the args.


```go
type RateLimitResult struct {
	Reset time.Duration
	Remaining int
	Limit int
}

func (r *RateLimitResult) Unwrap() (bool, error){
	if r.Remaining > 0 {
		return true, nil
	}

	return false, ErrRateLimited
}

function RateLimit(key string) (Resultable[bool], error)
	// The error will always be infra-related error.
 	return &RateLimitResult{}, nil
}


func main() {
	res, err := RateLimit("foo/bar")
	if err != nil {
		panic(err)
	}
	rl, ok := res.(*RateLimitResult)
	// Do sth ...
}
```
