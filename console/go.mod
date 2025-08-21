module github.com/olive-io/olive/console

go 1.23

replace (
	github.com/olive-io/olive/api => ../api
	github.com/olive-io/olive/clientgo => ../clientgo
	github.com/olive-io/olive/pkg => ../pkg
)
