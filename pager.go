package page

import (
	"context"
	"fmt"
	"sync"
)

//go:generate mockgen -source=pager.go -destination mocks_world_test.go -package page

const (
	defaultPageSize = 100
)

type (
	Loader[T any] interface {
		Load(ctx context.Context, pageKey string, pageSize int) (page []T, nextPageKey string, err error)
	}

	Pager[T any] struct {
		// elements count per page.
		pageSize int
		// next page key, that will be loaded in next call of Next.
		// 1. Loader will be called with this key until it return the empty next page key.
		// 2. First time Loader call will be done even if the next page key is empty.
		// 3. This field value will be updated after each call of Next.
		nextPageKey string
		// loader that loads the next page of elements.
		nextPageLoader Loader[T]
		// this field is used to check if the first page is loaded
		// to calculate IsAllLoaded result.
		isFirstPageLoaded bool
		// this field is used to ensure that isFirstPageLoaded is set only once.
		isFirstPageLoadedOnce sync.Once
	}
)

// New creates a new Pager.
func New[T any](opts ...Option[T]) (*Pager[T], error) {
	pager := &Pager[T]{
		pageSize:              defaultPageSize,
		isFirstPageLoaded:     false,
		isFirstPageLoadedOnce: sync.Once{},
	}

	for _, opt := range opts {
		if err := opt(pager); err != nil {
			return nil, err
		}
	}

	if pager.nextPageLoader == nil {
		return nil, fmt.Errorf("next page loader is required")
	}

	return pager, nil
}

// Next returns the next page of elements.
func (p *Pager[T]) Next(ctx context.Context) ([]T, error) {
	if p.IsAllLoaded() {
		return nil, nil
	}

	page, nextPageKey, err := p.nextPageLoader.Load(ctx, p.nextPageKey, p.pageSize)
	if err != nil {
		return nil, fmt.Errorf("page %s: %w", p.nextPageKey, err)
	}
	p.nextPageKey = nextPageKey
	p.pageLoadedAtLeastOnceTime()
	return page, nil
}

// All returns all elements from all pages.
func (p *Pager[T]) All(ctx context.Context) ([]T, error) {
	allPages := make([]T, 0, p.pageSize)
	for !p.IsAllLoaded() {
		page, err := p.Next(ctx)
		if err != nil {
			return allPages, err
		}
		allPages = append(allPages, page...)
	}
	return allPages, nil
}

func (p *Pager[T]) IsAllLoaded() bool {
	return p.nextPageKey == "" && p.isFirstPageLoaded
}

func (p *Pager[T]) pageLoadedAtLeastOnceTime() {
	p.isFirstPageLoadedOnce.Do(func() {
		p.isFirstPageLoaded = true
	})
}
