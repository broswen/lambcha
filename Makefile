
.PHONY: build clean deploy

build: clean 
	export GO111MODULE=on
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/generate_code generate_code/main.go
	env CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/validate_code validate_code/main.go
	cp generate_code/Blox2.ttf bin/Blox2.ttf

clean:
	rm -rf ./bin 

deploy: clean build
	sls deploy --verbose