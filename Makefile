build:
	go build -o bin/locksmith ./

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
