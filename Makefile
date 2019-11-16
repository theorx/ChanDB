clear:
	rm data/*.txt
	rm bench/*.txt
	rm bench/cache/*.txt
build:
	go build -o bin_bench bench.go
benchmark:
	/usr/bin/time -v ./bin_bench
run:
	go run -race bench.go
