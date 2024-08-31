.PHONY: rex clean

rex:
	@go build

clean:
	@rm rex

test: rex
	@go test -C test -v
