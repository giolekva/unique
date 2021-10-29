clean:
	rm -f operator

generate:
	rm -rf generated
	./hack/generate.sh

operator: clean
	go mod tidy
	go mod vendor
	go build -o operator *.go
