default:
	go build -o rna main.go

fmt:
	find main.go src -type f -exec gofmt -w {} \;

test:
	go test -v rna/packet
