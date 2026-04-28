APP_NAME=scaly
build:
	go build -o $(APP_NAME)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME).exe

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o $(APP_NAME)-mac

run:
	go run .

clean:
	rm -f $(APP_NAME).exe $(APP_NAME)-linux $(APP_NAME)-mac