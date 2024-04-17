module github.com/olive-io/olive

go 1.21

replace (
	github.com/olive-io/olive/api => ./api
	github.com/olive-io/olive/client => ./client
	github.com/olive-io/olive/cmd => ./cmd
	github.com/olive-io/olive/gateway => ./gateway
	github.com/olive-io/olive/meta => ./meta
	github.com/olive-io/olive/pkg => ./pkg
	github.com/olive-io/olive/runner => ./runner
)
