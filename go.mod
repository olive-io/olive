module github.com/olive-io/olive

go 1.23

replace (
	github.com/olive-io/olive/api => ./api
	github.com/olive-io/olive/client => ./client
	github.com/olive-io/olive/pkg => ./pkg
	github.com/olive-io/olive/mon => ./mon
)

