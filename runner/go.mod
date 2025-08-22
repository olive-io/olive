module github.com/olive-io/olive/runner

go 1.23.7

replace (
	github.com/olive-io/olive/api => ../api
	github.com/olive-io/olive/clientgo => ../clientgo
	github.com/olive-io/olive/pkg => ../pkg
)
