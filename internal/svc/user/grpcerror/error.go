package grpcerror

import (
	"github.com/hailsayan/achilles/internal/svc/user/constant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUserNotFoundError() error {
	return status.Error(codes.NotFound, constant.UserNotFoundErrorMessage)
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

func NewCacheSetError() error {
	return status.Error(codes.Internal, constant.CacheSetError)
}

func NewCacheDeleteError() error {
	return status.Error(codes.Internal, constant.CacheDeleteError)
}
