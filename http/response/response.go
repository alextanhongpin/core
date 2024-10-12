package response

type Error struct {
	Code             string           `json:"code"`
	Message          string           `json:"message"`
	ValidationErrors ValidationErrors `json:"validationErrors,omitempty"`
}

type Body struct {
	Data     any       `json:"data,omitempty"`
	Error    *Error    `json:"error,omitempty"`
	PageInfo *PageInfo `json:"pageInfo,omitempty"`
}

type PageInfo struct {
	HasPrevPage bool   `json:"hasPrevPage"`
	HasNextPage bool   `json:"hasNextPage"`
	StartCursor string `json:"startCursor"`
	EndCursor   string `json:"endCursor"`
}
