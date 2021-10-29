clean:
	rm -f server_*

server_worker: cmd/worker.go rpc/* worker/*
	go build -o server_worker cmd/worker.go

server_controller: cmd/controller.go rpc/* controller/*
	go build -o server_controller cmd/controller.go

image: clean server_worker server_controller
	docker build --tag=giolekva/unique:v5 . --platform=linux/arm64

push: export GOOS=linux
push: export GOARCH=arm64
push: export CGO_ENABLED=0
push: export GO111MODULE=on
push: image
	docker push giolekva/unique:v5
