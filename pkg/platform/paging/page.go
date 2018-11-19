package paging

import (
	"context"
)

type Pager interface {
	// GetResult should return the current fetched results.
	// In case Next() is called it should return the next result if they are fetched
	GetResult() interface{}

	// Next should try to fetch the next result and when succeed,
	// Result() should return the fetched result
	// TODO: May be next can return boolean to show whether there are more pages
	Next(context.Context) error

	HasNext() bool
}
