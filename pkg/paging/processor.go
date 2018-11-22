package paging

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
)

type ProcessFunc func(interface{}) error

type PageProcessor struct {
	Pager Pager
}

func (p *PageProcessor) Process(ctx context.Context, fn ProcessFunc) error {
	for {
		result := p.Pager.GetResult()
		err := fn(result)
		if err != nil {
			return err
		}

		if !p.Pager.HasNext() {
			break
		}

		err = p.Pager.Next(ctx)
		if err != nil {
			// TODO: Add retry logic here
			return fmt.Errorf("error during page fetch: %s", err.Error())
		}

		select {
		case <-ctx.Done():
			log.C(ctx).Info("Page processing cancelled")
			return nil

		default:
		}
	}

	return nil
}
