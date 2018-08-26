# horizon

### introduce
smart high performance memory cache with high concurrent.

horizon has much better performance and protects upstreams when refreshing.

you can visit remote data not change often just like to visit local data.

### benchmark
goos: darwin

goarch: amd64

pkg: github.com/qingyu31/horizon/benchmark

BenchmarkHorizon-4

1000000	       538 ns/op

max time cost per request less then 1 millisecond
