package rule

import (
    "strings"
    "go/parser"
    "go/token"
    "testing"

    "github.com/mgechev/revive/lint"
)

// TestErrorStringsLint constructs a source code file containing error strings,
func TestErrorStringsLint(t *testing.T) {
    src := `
        package test
        import (
            "errors"
            "fmt"
        )
        func f() error {
            return errors.New("Bad error.")
        }
        func g() error {
            return fmt.Errorf("bad error")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source code: %v", err)
    }
    file := &lint.File{
        AST: fileAst,
    }
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure ErrorStringsRule: %v", err)
    }
    failures := rule.Apply(file, nil)
    // Expect a failure for errors.New("Bad error.") because the error string is capitalized and ends with punctuation.
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure, got %d", len(failures))
    }
    expectedMsg := "error strings should not be capitalized or end with punctuation or a newline"
    if failures[0].Failure != expectedMsg {
        t.Errorf("unexpected failure message: got '%s', want '%s'", failures[0].Failure, expectedMsg)
    }
}

// TestConfigureInvalidCustomFunction tests that configuring the rule with an invalid custom function returns an error.
func TestConfigureInvalidCustomFunction(t *testing.T) {
    rule := &ErrorStringsRule{}
    err := rule.Configure(lint.Arguments{"invalidCustomFunctionName"})
    if err == nil {
        t.Fatal("expected an error when configuring an invalid custom function, got nil")
    }
}
// TestCleanAndEdgeCases tests various error string scenarios including empty, clean and violation details.
func TestCleanAndEdgeCases(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            // Should not report: empty string.
            _ = errors.New("")
            // Should not report: clean message with lowercase initial.
            _ = errors.New("bad error")
            // Should report: capitalized error message.
            _ = errors.New("Bad error")
            // Should report: error string ending with punctuation.
            _ = errors.New("bad error.")
            // Should not report: error message with uppercase initialism.
            _ = errors.New("FOO")
            return nil
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    // Expect 2 failures: one for "Bad error" and one for "bad error.".
    if len(failures) != 2 {
        t.Fatalf("expected 2 failures, got %d", len(failures))
    }
}

// TestMessageFromSecondArg tests that the error message is correctly extracted from the second argument
// when the first argument is not a string literal.
func TestMessageFromSecondArg(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            var x int
            // First argument is not a basic string literal; should use the second argument for the error message.
            return errors.New(x, "Bad error")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure, got %d", len(failures))
    }
}

// TestCustomErrorFunction tests that a custom error function provided by the user is correctly linted.
func TestCustomErrorFunction(t *testing.T) {
    src := `
        package test
        import "custom"
        func f() error {
            return custom.CustomError("Bad error")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    // Configure the rule with a custom function mapping.
    if err := rule.Configure(lint.Arguments{"custom.CustomError"}); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure, got %d", len(failures))
    }
}

// TestNonErrorFunctionCall tests that function calls that do not match any known error function are ignored.
func TestNonErrorFunctionCall(t *testing.T) {
    src := `
        package test
        func f() error {
            // This function call does not match any configured error function.
            return someFunc("Bad error")
        }
        func someFunc(s string) error {
            return nil
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures, got %d", len(failures))
    }
}
// TestLintErrorString tests the lintErrorString function directly with various inputs.
func TestLintErrorString(t *testing.T) {
    tests := []struct{
        input         string
        expectedClean bool
        expectedConf  float64
    }{
        {"", true, 0},
        {"bad error", true, 0},
        {"Bad error", false, 0.6},
        {"bad error.", false, 0.8},
        {"B", false, 0.6},
        {"FOO", true, 0},
        {"UI error", true, 0},
    }
    for _, tc := range tests {
        clean, conf := lintErrorString(tc.input)
        if clean != tc.expectedClean || conf != tc.expectedConf {
            t.Errorf("lintErrorString(%q) = (%v, %v), expected (%v, %v)", tc.input, clean, conf, tc.expectedClean, tc.expectedConf)
        }
    }
}

// TestNonStringLiteralError tests that error function calls with no basic string literal are ignored.
func TestNonStringLiteralError(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            x := 42
            y := "Bad error" // Note: y is an identifier, not a basic literal.
            return errors.New(x, y)
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures, got %d", len(failures))
    }
}
// TestConfigureNonStringArg tests that non-string arguments in Configure are ignored.
func TestConfigureNonStringArg(t *testing.T) {
    rule := &ErrorStringsRule{}
    // Non-string arguments should be ignored, so we expect Configure to succeed.
    if err := rule.Configure(lint.Arguments{123, false}); err != nil {
        t.Fatalf("expected nil error when configuring with non-string arguments, got %v", err)
    }
}

// TestCallExprNoArgs tests that a call expression with no arguments does not produce a failure.
func TestCallExprNoArgs(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            // Call with no arguments: rule should ignore this call.
            return errors.New()
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures for call with no arguments, got %d", len(failures))
    }
}

// TestErrorStringEndingNewline tests that error strings ending with a newline trigger a failure.
func TestErrorStringEndingNewline(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            return errors.New("bad error\n")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure for error string ending with newline, got %d", len(failures))
    }
    expectedMsg := "error strings should not be capitalized or end with punctuation or a newline"
    if failures[0].Failure != expectedMsg {
        t.Errorf("unexpected failure message: got '%s', want '%s'", failures[0].Failure, expectedMsg)
    }
}

// TestLintErrorStringNonASCII tests lintErrorString with non-ASCII characters.
func TestLintErrorStringNonASCII(t *testing.T) {
    tests := []struct{
        input         string
        expectedClean bool
        expectedConf  float64
    }{
        {"érror", true, 0},  // lower-case accented letter: should be clean.
        {"Érror", false, 0.6}, // upper-case accented letter: should trigger a warning with lower confidence.
    }
    for _, tc := range tests {
        clean, conf := lintErrorString(tc.input)
        if clean != tc.expectedClean || conf != tc.expectedConf {
            t.Errorf("lintErrorString(%q) = (%v, %v), expected (%v, %v)", tc.input, clean, conf, tc.expectedClean, tc.expectedConf)
        }
    }
}
// TestErrorsWrap tests errors.Wrap function call with a violation in the error string.
func TestErrorsWrap(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            return errors.Wrap("Bad error", nil)
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure, got %d", len(failures))
    }
}

// TestErrorsWithMessage tests errors.WithMessage with a violation in the error string.
func TestErrorsWithMessage(t *testing.T) {
    src := `
        package test
        import "errors"
        func f() error {
            return errors.WithMessage(nil, "Error occurred!")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 1 {
        t.Fatalf("expected 1 failure, got %d", len(failures))
    }
}

// TestFmtErrorfClean tests fmt.Errorf with a clean error string.
func TestFmtErrorfClean(t *testing.T) {
    src := `
        package test
        import "fmt"
        func f() error {
            return fmt.Errorf("good error")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures, got %d", len(failures))
    }
}

// TestCallExprWithNonBasicLiteralOneArg tests that a call expression with a non-basic literal as the only argument is ignored.
func TestCallExprWithNonBasicLiteralOneArg(t *testing.T) {
    src := `
        package test
        import "errors"
        func f(x string) error {
            return errors.New(x)
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures, got %d", len(failures))
    }
}

// TestConfigureMultipleValidCustomFunctions tests configuration with multiple valid custom functions.
func TestConfigureMultipleValidCustomFunctions(t *testing.T) {
    rule := &ErrorStringsRule{}
    err := rule.Configure(lint.Arguments{"custom.CustomError", "another.AnotherError"})
    if err != nil {
        t.Fatalf("unexpected error configuring valid custom functions: %v", err)
    }
    if _, ok := rule.errorFunctions["custom"]; !ok {
        t.Errorf("expected custom package to be configured")
    }
    if _, ok := rule.errorFunctions["another"]; !ok {
        t.Errorf("expected another package to be configured")
    }
}

// TestConfigureEmptyArguments tests that configuring with empty arguments does not fail.
func TestConfigureEmptyArguments(t *testing.T) {
    rule := &ErrorStringsRule{}
    err := rule.Configure(lint.Arguments{})
    if err != nil {
        t.Fatalf("expected nil error with empty arguments, got: %v", err)
    }
}
// TestConfigureMixedCustomFunctions tests that a mixture of valid and invalid custom function arguments causes an error.
func TestConfigureMixedCustomFunctions(t *testing.T) {
    rule := &ErrorStringsRule{}
    // "custom.CustomError" is valid, but "invalidFormat" is not valid (missing a dot separator).
    err := rule.Configure(lint.Arguments{"custom.CustomError", "invalidFormat"})
    if err == nil {
        t.Fatal("expected an error when configuring with mixed valid and invalid custom functions, got nil")
    }
    if !strings.Contains(err.Error(), "invalidFormat") {
        t.Fatalf("unexpected error message, got: %v", err)
    }
}

// TestCallExprNonSelector tests that call expressions that are not selector expressions are ignored.
func TestCallExprNonSelector(t *testing.T) {
    // Create a source where the function being called is an identifier (not a selector).
    src := `
        package test
        func New(msg string) error { return nil }
        func f() error {
            // Here New is called as an identifier, not as pkg.New so it should not be linted.
            return New("Bad error")
        }
    `
    fset := token.NewFileSet()
    fileAst, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
    if err != nil {
        t.Fatalf("failed to parse source: %v", err)
    }
    file := &lint.File{AST: fileAst}
    rule := &ErrorStringsRule{}
    if err := rule.Configure(nil); err != nil {
        t.Fatalf("failed to configure rule: %v", err)
    }
    failures := rule.Apply(file, nil)
    // No failure should be reported because the function call does not match a selector expression.
    if len(failures) != 0 {
        t.Fatalf("expected 0 failures for call expression without a selector, got %d", len(failures))
    }
}

// TestLintErrorStringPunctuation tests error strings ending with a colon or exclamation.
func TestLintErrorStringPunctuation(t *testing.T) {
    // Test a string ending with ":".
    clean, conf := lintErrorString("Bad error:")
    if clean || conf != 0.8 {
        t.Errorf("lintErrorString(%q) = (%v, %v), expected (false, 0.8)", "Bad error:", clean, conf)
    }

    // Test a string ending with "!".
    clean, conf = lintErrorString("Bad error!")
    if clean || conf != 0.8 {
        t.Errorf("lintErrorString(%q) = (%v, %v), expected (false, 0.8)", "Bad error!", clean, conf)
    }
}