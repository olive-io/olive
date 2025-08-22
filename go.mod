module github.com/olive-io/olive

go 1.23.7

replace (
	github.com/olive-io/olive/api => ./api
	github.com/olive-io/olive/clientgo => ./clientgo
	github.com/olive-io/olive/console => ./console
	github.com/olive-io/olive/pkg => ./pkg
	github.com/olive-io/olive/runner => ./runner
	github.com/olive-io/olive/server => ./server
)
