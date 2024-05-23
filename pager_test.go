package page

import (
	"context"
	"errors"
	"go.uber.org/mock/gomock"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	loader := NewMockLoader[[]int](ctrl)
	cases := []struct {
		name    string
		opts    []Option[[]int]
		want    *Pager[[]int]
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "ok, with options",
			opts: []Option[[]int]{
				WithPageSize[[]int](20),
				WithNextPageKey[[]int]("next-page-key"),
				WithNextPageLoader[[]int](loader),
			},
			want: &Pager[[]int]{
				pageSize:              20,
				nextPageKey:           "next-page-key",
				nextPageLoader:        loader,
				isFirstPageLoaded:     false,
				isFirstPageLoadedOnce: sync.Once{},
			},
			wantErr: require.NoError,
		},
		{
			name: "no next page loader",
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "next page loader is required")
			},
		},
		{
			name: "nil next page loader",
			opts: []Option[[]int]{
				WithNextPageLoader[[]int](nil),
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "next page loader is required")
			},
		},
		{
			name: "empty next page key in options",
			opts: []Option[[]int]{
				WithNextPageLoader[[]int](loader),
				WithNextPageKey[[]int](""),
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "next page key must not be empty")
			},
		},
		{
			name: "zero page size",
			opts: []Option[[]int]{
				WithNextPageLoader[[]int](loader),
				WithPageSize[[]int](0),
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "page size must be positive")
			},
		},
		{
			name: "negative page size",
			opts: []Option[[]int]{
				WithNextPageLoader[[]int](loader),
				WithPageSize[[]int](-1),
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "page size must be positive")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := New(tc.opts...)
			tc.wantErr(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPager_Next(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		name            string
		loader          func() Loader[int]
		opts            []Option[int]
		want            []int
		wantErr         require.ErrorAssertionFunc
		wantNextPageKey string
	}{
		{
			name: "all ok, default options",
			loader: func() Loader[int] {
				loader := NewMockLoader[int](ctrl)
				loader.EXPECT().
					Load(gomock.Any(), "", defaultPageSize).
					Times(1).
					Return([]int{1, 2, 3}, "next", nil)
				return loader
			},
			want:            []int{1, 2, 3},
			wantErr:         require.NoError,
			wantNextPageKey: "next",
		},
		{
			name: "loader returned empty page",
			loader: func() Loader[int] {
				loader := NewMockLoader[int](ctrl)
				loader.EXPECT().
					Load(gomock.Any(), "", defaultPageSize).
					Times(1).
					Return([]int{}, "next", nil)
				return loader
			},
			want:            []int{},
			wantErr:         require.NoError,
			wantNextPageKey: "next",
		},
		{
			name: "loader returned an error",
			loader: func() Loader[int] {
				loader := NewMockLoader[int](ctrl)
				loader.EXPECT().
					Load(gomock.Any(), "previous-page-key", defaultPageSize).
					Times(1).
					Return(nil, "", errors.New("some error"))
				return loader
			},
			opts: []Option[int]{
				WithNextPageKey[int]("previous-page-key"),
			},
			wantNextPageKey: "previous-page-key",
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "page previous-page-key: some error")
			},
		},
		{
			name: "start with custom next page key",
			loader: func() Loader[int] {
				loader := NewMockLoader[int](ctrl)
				loader.EXPECT().
					Load(gomock.Any(), "custom-page-key", defaultPageSize).
					Times(1).
					Return([]int{1, 2}, "next-page-key", nil)
				return loader
			},
			opts: []Option[int]{
				WithNextPageKey[int]("custom-page-key"),
			},
			want:            []int{1, 2},
			wantNextPageKey: "next-page-key",
			wantErr:         require.NoError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts = append(tc.opts, WithNextPageLoader[int](tc.loader()))
			pager, err := New[int](tc.opts...)
			require.NoError(t, err)

			got, err := pager.Next(context.Background())
			tc.wantErr(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantNextPageKey, pager.nextPageKey)
			if err == nil {
				require.True(t, pager.isFirstPageLoaded)
			}
		})
	}
}

func TestPager_All(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cases := []struct {
		name    string
		loader  func() Loader[int]
		opts    []Option[int]
		want    []int
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "get 3 pages",
			loader: func() Loader[int] {
				loader := &fakeLoader{
					pageByKey: map[string][]int{
						"":                {1, 2, 3},
						"second-page-key": {4, 5, 6},
						"third-page-key":  {7, 8},
					},
					nextPageKeyByPageKey: map[string]string{
						"":                "second-page-key",
						"second-page-key": "third-page-key",
						"third-page-key":  "",
					},
					errOnPageKey: "no-error",
				}
				return loader
			},
			opts: []Option[int]{
				WithPageSize[int](3),
			},
			want:    []int{1, 2, 3, 4, 5, 6, 7, 8},
			wantErr: require.NoError,
		},
		{
			name: "get 2 pages, an error at the third page",
			loader: func() Loader[int] {
				loader := &fakeLoader{
					pageByKey: map[string][]int{
						"":                {1, 2, 3},
						"second-page-key": {4, 5, 6},
						"third-page-key":  {7, 8},
					},
					nextPageKeyByPageKey: map[string]string{
						"":                "second-page-key",
						"second-page-key": "third-page-key",
						"third-page-key":  "",
					},
					errOnPageKey: "third-page-key",
				}
				return loader
			},
			opts: []Option[int]{
				WithPageSize[int](3),
			},
			want: []int{1, 2, 3, 4, 5, 6},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "page third-page-key: test error")
			},
		},
		{
			name: "get 1 page, an error at the second page",
			loader: func() Loader[int] {
				loader := &fakeLoader{
					pageByKey: map[string][]int{
						"":                {1, 2, 3},
						"second-page-key": {4, 5, 6},
						"third-page-key":  {7, 8},
					},
					nextPageKeyByPageKey: map[string]string{
						"":                "second-page-key",
						"second-page-key": "third-page-key",
						"third-page-key":  "",
					},
					errOnPageKey: "second-page-key",
				}
				return loader
			},
			opts: []Option[int]{
				WithPageSize[int](3),
			},
			want: []int{1, 2, 3},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.EqualError(t, err, "page second-page-key: test error")
			},
		},
		{
			name: "nothing to load",
			loader: func() Loader[int] {
				loader := NewMockLoader[int](ctrl)
				loader.EXPECT().
					Load(gomock.Any(), "", defaultPageSize).
					Times(1).
					Return([]int{}, "", nil)
				return loader
			},
			want:    []int{},
			wantErr: require.NoError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.opts = append(tc.opts, WithNextPageLoader[int](tc.loader()))
			pager, err := New[int](tc.opts...)
			require.NoError(t, err)

			got, err := pager.All(context.Background())
			tc.wantErr(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

type fakeLoader struct {
	pageByKey            map[string][]int
	nextPageKeyByPageKey map[string]string
	errOnPageKey         string
}

func (l *fakeLoader) Load(_ context.Context, pageKey string, _ int) (page []int, nextPageKay string, err error) {
	if l.errOnPageKey == pageKey {
		return nil, "", errors.New("test error")
	}
	return l.pageByKey[pageKey], l.nextPageKeyByPageKey[pageKey], nil
}
