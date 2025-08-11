package errors

import (
	"errors"
	"fmt"
)

type BaseError struct {
	Err error
}

func (e *BaseError) Error() string {
	return fmt.Sprintf("%s", e.Err)
}

func (e *BaseError) Cause() error {
	return e.Err
}

func (e *BaseError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

type Internal struct {
	BaseError
}

func NewInternal(err error) *Internal {
	return &Internal{BaseError: BaseError{Err: err}}
}
func (e Internal) Internal() {}

type BadRequest struct {
	BaseError
}

func NewBadRequest(err error) *BadRequest {
	return &BadRequest{BaseError: BaseError{Err: err}}
}
func (e BadRequest) BadRequest() {}

type NotFoundRequest struct {
	BaseError
}

func NewNotFound(err error) *NotFoundRequest {
	return &NotFoundRequest{BaseError: BaseError{Err: err}}
}
func (e NotFoundRequest) NotFoundRequest() {}

type ConflictRequest struct {
	BaseError
}

func NewConflict(err error) *ConflictRequest {
	return &ConflictRequest{BaseError: BaseError{Err: err}}
}
func (e ConflictRequest) ConflictRequest() {}
