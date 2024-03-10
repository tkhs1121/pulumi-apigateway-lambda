.PHONY: build

build:
	cd cmd && GOOS=linux GOARCH=amd64 go build -ldflags=$(BUILD_LDFLAGS) -o ../build/bootstrap ./...
	cd build && zip ../function.zip ./bootstrap

clean:
	rm -rf function.zip ./build