package grpcerr

import (
	"fmt"

	"google.golang.org/grpc/status"
)

func (e ErrCode) Error() string {
	return fmt.Sprintf("grpc err code: %d", uint32(e))
}

func (e ErrCode) Code() uint32 {
	return uint32(e)
}

func (e ErrCode) Msg() string {
	if msg, ok := codeMsg[e]; ok {
		return msg
	}
	return ""
}

func ToError(code uint32) error {
	return ErrCode(code)
}

func (e ErrCode) Equal(status *status.Status) bool {
	if status == nil {
		return false
	}
	if uint32(status.Code()) != e.Code() {
		return false
	}
	return true
}
