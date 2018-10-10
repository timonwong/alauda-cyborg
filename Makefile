
dep:
	dep ensure -v

test:
	go test -v -tags unit -cover -coverprofile cover.out   ./pkg/client/
	go tool cover -html=cover.out -o coverage.html
	open coverage.html
