package errors

import (
	"errors"
	"log"
	"net/http"
)

type InternalErr interface {
	Internal()
}

type BadRequestErr interface {
	BadRequest()
}

type NotFoundErr interface {
	NotFoundRequest()
}

func logCode(code int, error error) {
	log.Printf("api log: %s, statusCode: %v", error.Error(), code)
}

func HandleError(h func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		// unwrap return nil if error is not wrapped
		wrappedErr := errors.Unwrap(err)
		if wrappedErr == nil {
			wrappedErr = err
		}

		var internal InternalErr
		if errors.As(wrappedErr, &internal) {
			logCode(http.StatusInternalServerError, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var badRequest BadRequestErr
		if errors.As(wrappedErr, &badRequest) {
			logCode(http.StatusBadRequest, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var notFoundRequest NotFoundErr
		if errors.As(wrappedErr, &notFoundRequest) {
			logCode(http.StatusNotFound, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// unknown error
		logCode(http.StatusInternalServerError, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
