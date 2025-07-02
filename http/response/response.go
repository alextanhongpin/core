package response

// PageInfo contains pagination information
type PageInfo struct {
	HasPrevPage bool   `json:"hasPrevPage"`
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor,omitempty"`
	EndCursor   string `json:"endCursor,omitempty"`
}

// Body represents the standard API response structure
type Body struct {
	Data     any       `json:"data,omitempty"`
	Error    *Error    `json:"error,omitempty"`
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

// Error represents an API error
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Errors  map[string]any `json:"errors,omitempty"`
}
