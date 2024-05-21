package page

import "fmt"

type Option[T any] func(*Pager[T]) error

func WithNextPageKey[T any](key string) Option[T] {
	return func(p *Pager[T]) error {
		if key == "" {
			return fmt.Errorf("next page key must not be empty")
		}
		p.nextPageKey = key
		return nil
	}
}

func WithPageSize[T any](pageSize int) Option[T] {
	return func(p *Pager[T]) error {
		if pageSize <= 0 {
			return fmt.Errorf("page size must be positive")
		}
		p.pageSize = pageSize
		return nil
	}
}

func WithNextPageLoader[T any](loader Loader[T]) Option[T] {
	return func(p *Pager[T]) error {
		if loader == nil {
			return fmt.Errorf("next page loader is required")
		}
		p.nextPageLoader = loader
		return nil
	}
}
