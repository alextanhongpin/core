package requestid

import "github.com/alextanhongpin/core/http/contextkey"

var Context contextkey.Key[string] = "request_id"
