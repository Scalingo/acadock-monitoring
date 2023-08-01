package handlers

import (
	"fmt"
	"strings"
)

type BadRequestError struct {
	Errors map[string][]string `json:"errors"`
}

func (err BadRequestError) Error() string {
	errArray := make([]string, 0, len(err.Errors))
	for errTitle, errValues := range err.Errors {
		errArray = append(errArray, fmt.Sprintf("* %s → %s", errTitle, strings.Join(errValues, ", ")))
	}
	return strings.Join(errArray, "\n")
}

func NewBadRequestErrors() *BadRequestError {
	return &BadRequestError{
		Errors: make(map[string][]string),
	}
}
