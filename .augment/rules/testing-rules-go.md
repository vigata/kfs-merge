---
type: "always_apply"
---

# Go Testing Rules

## Test Structure and Organization

- **File Naming:** Test files must be named `{source_file}_{differentiator}_test.go` where:
  - `{source_file}` is the name of the source file being tested (without `.go` extension)
  - `{differentiator}` describes the aspect or category of tests
  - Examples: `api_basic_test.go`, `strategies_array_test.go`, `schema_ref_test.go`, `validator_test.go`
  - For simple cases with a single test file per source, use `{source_file}_test.go` (e.g., `validator_test.go`)
- **Function Naming:** Test functions must start with `Test` followed by the name of the function or behavior being tested: `TestFunctionName` or `TestTypeName_MethodName`.
- **Same Package:** Place tests in the same package to test unexported functions. Use `_test` suffix package (e.g., `package foo_test`) for black-box testing of exported API only.
- **Testdata Directory:** Use a `testdata/` directory for test fixtures. Go tooling ignores this directory.

## Table-Driven Tests

- **Prefer Table-Driven Tests:** Use table-driven tests for testing multiple inputs and expected outputs.
- **Struct Slices:** Define test cases as a slice of anonymous structs with descriptive field names.
- **Use Subtests:** Run each case with `t.Run(name, func(t *testing.T) { ... })` for clear output and selective test running.

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

## Test Principles

- **F.I.R.S.T. Principles:** Tests should be Fast, Independent, Repeatable, Self-Validating, and Timely.
- **Test One Thing:** Each test or subtest should verify one specific behavior.
- **Clear Failure Messages:** Use `t.Errorf()` with descriptive messages: `got X, want Y` format.
- **Arrange-Act-Assert:** Structure tests with setup, execution, and verification phases.

## Testing Utilities

- **t.Helper():** Call `t.Helper()` in test helper functions to improve error reporting.
- **t.Parallel():** Use `t.Parallel()` for tests that can run concurrently to speed up test execution.
- **t.Cleanup():** Use `t.Cleanup(func() { ... })` for teardown logic instead of `defer`.
- **t.Skip():** Use `t.Skip("reason")` to skip tests conditionally (e.g., for integration tests).
- **t.Fatal() vs t.Error():** Use `t.Fatal()` when continuing the test is pointless. Use `t.Error()` to report failures but continue.

## Testify (if using)

- **require vs assert:** Use `require` for conditions that must pass to continue. Use `assert` for non-fatal checks.
- **Common Assertions:** `assert.Equal`, `assert.NoError`, `assert.ErrorIs`, `assert.Nil`, `assert.NotNil`.

## Mocking and Test Doubles

- **Interface-Based Mocking:** Design code with interfaces to allow easy mocking.
- **Define Interfaces Locally:** Define the interface you need in the test file or package.
- **Avoid Over-Mocking:** Mock at boundaries (I/O, external services). Don't mock everything.

## Benchmarks

- **Benchmark Functions:** Name benchmarks `BenchmarkFunctionName`. Use `b *testing.B` parameter.
- **b.N Loop:** Always loop `b.N` times. The framework adjusts this for accurate measurements.
- **b.ResetTimer():** Call after expensive setup to exclude setup time from measurements.
- **b.ReportAllocs():** Call to include memory allocation stats in benchmark output.

```go
func BenchmarkAdd(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Add(2, 3)
    }
}
```

## Example Functions

- **Example Functions:** Use `ExampleFunctionName()` for testable documentation.
- **Output Comments:** Include `// Output:` comment to verify expected output.

```go
func ExampleAdd() {
    fmt.Println(Add(2, 3))
    // Output: 5
}
```

## Running Tests

- **Run All Tests:** `go test ./...`
- **Run with Race Detector:** `go test -race ./...` — always run this in CI.
- **Verbose Output:** `go test -v ./...`
- **Run Specific Test:** `go test -run TestName ./...`
- **Run Specific Subtest:** `go test -run TestName/subtest_name ./...`
- **Coverage:** `go test -cover ./...` or `go test -coverprofile=coverage.out ./...`
- **Benchmarks:** `go test -bench=. ./...`
- **Short Mode:** Use `testing.Short()` and `-short` flag to skip long-running tests.

## Integration and E2E Tests

- **Build Tags:** Use `//go:build integration` to separate integration tests.
- **Test Flags:** Define custom flags for configuration (e.g., database URLs).
- **TestMain:** Use `TestMain(m *testing.M)` for setup/teardown across all tests in a package.

```go
func TestMain(m *testing.M) {
    // Setup
    code := m.Run()
    // Teardown
    os.Exit(code)
}
```

## CI/CD Guidelines

- **Always Run:** `go test -race -cover ./...` in CI pipelines.
- **Fail Fast:** Configure CI to stop on first test failure for faster feedback.
- **Cache:** Go test results are cached by default. Use `-count=1` to disable caching if needed.