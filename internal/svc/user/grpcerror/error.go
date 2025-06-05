package grpcerror

import (
	"github.com/hailsayan/achilles/internal/svc/user/constant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUserNotFoundError() error {
	return status.Error(codes.NotFound, constant.UserNotFoundErrorMessage)
}

func NewUsernameExistsError() error {
	return status.Error(codes.AlreadyExists, constant.UsernameExistsErrorMessage)
}

func NewEmailExistsError() error {
	return status.Error(codes.AlreadyExists, constant.EmailExistsErrorMessage)
}

func NewInternalError() error {
	return status.Error(codes.Internal, constant.InternalServerErrorMessage)
}

func NewUnavailableError() error {
	return status.Error(codes.Unavailable, constant.ServiceUnavailableMessage)
}

func NewInternalErrorWithMessage(msg string) error {
	if msg == "" {
		msg = constant.InternalServerErrorMessage
	}
	return status.Error(codes.Internal, msg)
}

func NewUnavailableErrorWithMessage(msg string) error {
	if msg == "" {
		msg = constant.ServiceUnavailableMessage
	}
	return status.Error(codes.Unavailable, msg)
}