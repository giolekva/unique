clean:
	rm -f operator

generate:
	rm -rf generated
	./hack/generate.sh

operator: clean
	go mod tidy
	go mod vendor
	go build -o operator *.go

image: operator
	docker build --tag=giolekva/unique-operator:v0.1 . --platform=linux/arm64

push: export GOOS=linux
push: export GOARCH=arm64
push: export CGO_ENABLED=0
push: export GO111MODULE=on
push: image
	docker push giolekva/unique-operator:v0.1
