package grpcdump

import (
	"google.golang.org/grpc/status"
)

type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewStatus(err error) *Status {
	sts, ok := status.FromError(err)
	if !ok {
		return nil
	}

	return &Status{
		Code:    sts.Code().String(),
		Message: sts.Message(),
	}
}
