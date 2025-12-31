---
type: "always_apply"
description: "Go code generation rules"
---

# Clean Code Rules for Go

## Foundational Principles

- **Clarity and Readability:** Generated code must be easily understandable. Prioritize clarity over cleverness.
- **Simplicity:** Generate the simplest possible code that solves the problem. Go favors explicit over implicit.
- **The Boy Scout Rule:** When modifying existing code, improve the overall quality of the codebase.
- **No Duplication (DRY):** Avoid redundant code. Use functions, methods, and packages to promote code reuse.

## Go Naming Conventions

- **MixedCaps:** Use `MixedCaps` or `mixedCaps` for multi-word names. Never use underscores.
- **Exported vs Unexported:** Uppercase first letter for exported (public), lowercase for unexported (private).
- **Short Names for Local Scope:** Use short names like `i`, `r`, `buf` for local variables with limited scope.
- **Longer Names for Larger Scope:** Use descriptive names for package-level declarations and exported identifiers.
- **Acronyms:** Keep acronyms uppercase: `HTTP`, `URL`, `ID`, `API` (e.g., `userID`, `HTTPClient`).
- **Interface Names:** Single-method interfaces use the method name plus `-er` suffix: `Reader`, `Writer`, `Stringer`.
- **Getters:** Don't use `Get` prefix. Use `obj.Name()` not `obj.GetName()`. Setters use `Set` prefix: `obj.SetName()`.
- **Package Names:** Lowercase, single-word, no underscores or mixedCaps. Avoid generic names like `util`, `common`.

## Functions and Methods

- **Small and Focused:** Functions should be small with a single, well-defined responsibility.
- **Do One Thing:** Each function should perform a single task and do it well.
- **Return Early:** Use guard clauses and early returns to reduce nesting.
- **Named Return Values:** Use sparingly, mainly for documentation. Initialize explicitly when used.
- **Receiver Names:** Use short, consistent receiver names (1-2 letters). Never use `this` or `self`.
- **Value vs Pointer Receivers:** Use pointer receivers for mutations or large structs. Be consistent within a type.

## Comments and Documentation

- **Godoc Format:** Doc comments start with the name being documented: `// FunctionName does X.`
- **Package Comments:** Every package should have a package comment. Use `doc.go` for lengthy documentation.
- **Complete Sentences:** Doc comments should be complete sentences ending with a period.
- **Explain Why, Not What:** Use comments to explain intent, not to restate code.
- **No Redundant Comments:** Avoid comments that simply describe what is obvious from the code.

## Formatting

- **Use gofmt:** Always format code with `gofmt` or `goimports`. Formatting is not a style choice in Go.
- **Imports:** Group imports: standard library, then blank line, then third-party packages.
- **Line Length:** Keep lines reasonable but don't obsess. `gofmt` handles most formatting.

## Structs and Interfaces

- **Accept Interfaces, Return Structs:** Functions should accept interface parameters and return concrete types.
- **Small Interfaces:** Prefer small interfaces with 1-2 methods. Compose larger interfaces from smaller ones.
- **Define Interfaces at Point of Use:** Define interfaces in the package that uses them, not the package that implements them.
- **Embedding:** Use struct embedding for composition. Avoid deep embedding hierarchies.
- **Zero Values:** Design structs so their zero value is useful and valid.
- **Constructor Functions:** Use `NewTypeName()` for constructors when zero value isn't sufficient.

## Error Handling

- **Errors Are Values:** Return errors as the last return value. Check errors immediately after the call.
- **Handle or Return:** Either handle the error or return it. Don't ignore errors silently.
- **Add Context:** Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`.
- **Custom Error Types:** Create custom error types for errors that need to be inspected programmatically.
- **errors.Is and errors.As:** Use `errors.Is()` for comparison and `errors.As()` for type assertion.
- **Don't Panic:** Reserve `panic` for truly unrecoverable situations. Use errors for expected failure cases.
- **Sentinel Errors:** Define package-level error variables for well-known errors: `var ErrNotFound = errors.New("not found")`.

## Boundaries and Dependencies

- **Isolate Third-Party Code:** Create wrapper packages or adapters to isolate external dependencies.
- **Dependency Injection:** Pass dependencies as function/method parameters or struct fields.
- **Interface Segregation:** Define minimal interfaces at boundaries to reduce coupling.

## Package Design

- **Separation of Concerns:** Each package should have a clear, focused purpose.
- **Avoid Circular Dependencies:** Design packages to avoid import cycles.
- **Internal Packages:** Use `internal/` directory to hide implementation details from external consumers.
- **Cmd Pattern:** Use `cmd/` directory for main packages in multi-binary projects.

## Simple Design

- **Go Proverb: "Clear is better than clever."**
- **Runs All Tests:** Code must work correctly and be verifiable through tests.
- **No Duplicate Code:** Eliminate redundancy. Extract common code into functions or packages.
- **Expresses Intent:** Code should clearly communicate what it does.
- **Minimal Abstractions:** Don't over-engineer. Add abstractions only when needed.

## Concurrency

- **Share Memory by Communicating:** Use channels to pass data between goroutines.
- **Goroutines Are Cheap:** Don't hesitate to spawn goroutines, but manage their lifecycle.
- **Use context.Context:** Pass `context.Context` as the first parameter for cancellation and timeouts.
- **sync Package:** Use `sync.Mutex`, `sync.RWMutex`, `sync.WaitGroup`, `sync.Once` when channels aren't appropriate.
- **Avoid Data Races:** Use `go run -race` to detect race conditions. Never ignore race warnings.
- **Channel Direction:** Specify channel direction in function signatures: `chan<-` (send-only), `<-chan` (receive-only).
- **Close Channels:** Only the sender should close a channel. Never close a receive-only channel.
- **Select for Multiple Channels:** Use `select` to wait on multiple channel operations.

## Refactoring and Code Smells

- **Identify Go-Specific Code Smells:**
  - Ignored errors (especially `_ = someFunc()`)
  - Naked returns in long functions
  - Over-use of `interface{}`/`any`
  - Deep nesting (prefer guard clauses)
  - Large parameter lists (use option structs)
  - Global mutable state
- **Continuous Improvement:** Regularly refactor to improve structure and readability.
- **Incremental Changes:** Make small, incremental improvements.

## Testing

- **Table-Driven Tests:** Use table-driven tests for testing multiple cases.
- **Test File Naming:** Tests go in `*_test.go` files in the same package.
- **Test Function Naming:** Use `TestFunctionName` or `TestTypeName_MethodName`.
- **Subtests:** Use `t.Run()` for subtests to group related test cases.
- **Testable Code:** Design for testability. Use interfaces for dependencies.
- **Example Functions:** Use `Example` functions for documentation and verification.
- **Benchmarks:** Use `Benchmark` functions for performance testing.

## AI-Specific Guidelines for Go Code Generation

- **Context Awareness:** Match the existing codebase style and patterns.
- **Idiomatic Go:** Always generate idiomatic Go code. Follow Effective Go guidelines.
- **Security:** Validate inputs, avoid hardcoded secrets, use `crypto/rand` not `math/rand` for security.
- **Performance:** Prefer stack allocation, minimize allocations in hot paths, use `sync.Pool` for frequently allocated objects.
- **Error Messages:** Use lowercase, no punctuation at the end: `fmt.Errorf("failed to open file")`.
- **Backward Compatibility:** Maintain API compatibility unless explicitly requested otherwise.
- **Testability:** Structure code for easy testing with dependency injection.

## Implementation Guidelines

### Code Generation Strategy

- **Start Simple:** Begin with the simplest solution that works, then refactor if needed.
- **Composition Over Inheritance:** Go has no inheritance. Use embedding and interfaces for composition.
- **Fail Fast:** Return errors early. Use guard clauses to reduce nesting.
- **Explicit Over Implicit:** Go favors explicit code. Avoid magic or hidden behavior.

### Quality Assurance

- **Run go vet:** Ensure generated code passes `go vet` checks.
- **Run staticcheck:** Use `staticcheck` for additional static analysis.
- **golangci-lint:** Run `golangci-lint` for comprehensive linting.
- **gofmt/goimports:** Always format code. Use `goimports` to manage imports.

### Go Modules

- **Use Go Modules:** Always use Go modules for dependency management.
- **Semantic Versioning:** Follow semver for versioned modules.
- **Minimal Dependencies:** Minimize external dependencies. Prefer the standard library.
- **Vendor Directory:** Consider vendoring for reproducible builds in production.

---