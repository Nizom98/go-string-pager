# String pager

Go-String-Pager is a paginator library for Go.
It helps you to paginate your data in a simple way.

## Usage

```go
package main

type MyLoader struct{}

func main() {
	myLoader := &MyLoader{}

	pager, _ := page.New[int](
		page.WithNextPageLoader[int](myLoader),
	)
	result, _ := pager.All(context.Background())
	fmt.Println(result)
}

func (l *MyLoader) Load(
	_ context.Context,
	pageKey string,
	pageSize int,
) (pageResult []int, nextPageKey int, err error) {
	body := map[string]interface{}{
		"limit":    pageSize,
		"page_key": pageKey,
	}
	// TODO: write your own logic to load page result and new page key
	return pageResult, nextPageKey, nil
}

```