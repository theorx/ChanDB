clear:
	rm data/*.txt
build:
	go build -o bin_bench bench.go
benchmark:
	/usr/bin/time -v ./bin_bench
