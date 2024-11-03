lint:
	golangci-lint run

go-tidy:
	go work sync
	go mod tidy
	# Loop through all directories containing a go.mod file and run go mod tidy in each
	find . -type f -name "go.mod" -execdir go mod tidy \;

pre-commit:
	go-tidy
	lint
