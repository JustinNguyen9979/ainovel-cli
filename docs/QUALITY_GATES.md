# Quality gates

Mọi pull request vào `main` phải vượt các gate sau trước khi được merge hoặc phát hành:

1. `gofmt` không trả về file nào.
2. `go mod verify`, `go vet ./...`, toàn bộ Go tests và build CLI pass.
3. Race test cho các package stateful pass.
4. Tổng Go line coverage đạt tối thiểu 80%.
5. Benchmark hot path không chậm hơn baseline quá 5%.
6. `npm ci`, `npm test`, package allowlist và `npm pack` pass.
7. `govulncheck`, OSV scan, Gitleaks, npm audit và license policy pass.
8. Native validation pass trên Linux/macOS/Windows x64 và arm64 khi runner tương ứng sẵn sàng.
9. Artifact release có đủ archive, checksum, manifest và signature/provenance.

## Local commands

```bash
go mod verify
go vet ./...
go test -buildvcs=false -count=1 ./...
go test -race -buildvcs=false -count=1 ./internal/host ./internal/store ./internal/tools
go test -covermode=atomic -coverprofile=coverage.out ./...
COVERAGE_MINIMUM=80 scripts/ci/check-coverage.sh coverage.out
go test ./internal/utils -run '^$' -bench '^Benchmark(JSONFieldExtractor|StreamFilter)$' -benchmem -count=5 > /tmp/bench.txt
BENCHMARK_MAX_REGRESSION=5 scripts/ci/check-benchmark.sh benchmarks/baseline.json /tmp/bench.txt
npm ci
npm test
scripts/ci/verify-package.sh
```

Không chạy live LLM trong PR/release gate. Các eval bắt buộc phải dùng fixture/offline fake để không phát sinh chi phí, nondeterminism hoặc rò rỉ secret.

Coverage baseline hiện phải được nâng qua pull request trước khi bật production release. Benchmark baseline chỉ được thay đổi qua pull request có review và phải kèm lý do đo được.
