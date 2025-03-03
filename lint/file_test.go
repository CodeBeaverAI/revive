package lint

import (
    "go/ast"
    "go/token"
    "testing"
)

func TestFile_disabledIntervals(t *testing.T) {
    buildCommentGroups := func(comments ...string) []*ast.CommentGroup {
    commentGroups := make([]*ast.CommentGroup, 0, len(comments))
    for _, c := range comments {
    commentGroups = append(commentGroups, &ast.CommentGroup{
    List: []*ast.Comment{
        {Text: c},
    },
    })
    }
    return commentGroups
    }

    tests := []struct {
    name     string
    comments []*ast.CommentGroup
    expected disabledIntervalsMap
    }{
    {
    name:     "no directives",
    comments: buildCommentGroups("// some comment"),
    expected: disabledIntervalsMap{},
    },
    {
    name:     "disable rule",
    comments: buildCommentGroups("//revive:disable:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    },
    },
    {
    name:     "enable rule",
    comments: buildCommentGroups("//revive:enable:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {},
    },
    },
    {
    name:     "disable and enable rule",
    comments: buildCommentGroups("//revive:disable:rule1", "//revive:enable:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        },
        },
    },
    },
    },
    {
    name:     "disable-line rule",
    comments: buildCommentGroups("//revive:disable-line:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        },
        },
    },
    },
    },
    {
    name:     "enable-line rule",
    comments: buildCommentGroups("//revive:enable-line:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    },
    },
    {
    name:     "disable-next-line rule",
    comments: buildCommentGroups("//revive:disable-next-line:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        Line:     1,
        },
        To: token.Position{
        Filename: "test.go",
        Line:     1,
        },
        },
    },
    },
    },
    {
    name:     "enable-next-line rule",
    comments: buildCommentGroups("//revive:enable-next-line:rule1"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        Line:     1,
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    },
    },
    }

    for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
    f := &File{
    Name: "test.go",
    Pkg: &Package{
        fset: token.NewFileSet(),
    },
    AST: &ast.File{
        Comments: tt.comments,
    },
    }
    got := f.disabledIntervals(nil, false, make(chan Failure, 10))
    if len(got) != len(tt.expected) {
    t.Errorf("disabledIntervals() = got %v, want %v", got, tt.expected)
    }
    for rule, intervals := range got {
    expectedIntervals, ok := tt.expected[rule]
    if !ok {
        t.Errorf("unexpected rule %q", rule)
        continue
    }
    if len(intervals) != len(expectedIntervals) {
        t.Errorf("intervals for rule %q = got %+v, want %+v", rule, intervals, expectedIntervals)
        continue
    }
    for i, interval := range intervals {
        if interval != expectedIntervals[i] {
        t.Errorf("interval %d for rule %q = got %+v, want %+v", i, rule, interval, expectedIntervals[i])
        }
    }
    }
    })
    }
}

func TestFile_IsTest(t *testing.T) {
    tests := []struct {
    name     string
    fileName string
    want     bool
    }{
    {
    name:     "test file",
    fileName: "something_test.go",
    want:     true,
    },
    {
    name:     "non-test file",
    fileName: "something.go",
    want:     false,
    },
    {
    name:     "file with test in middle",
    fileName: "test_helper.go",
    want:     false,
    },
    }

    for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
    f := &File{
    Name: tt.fileName,
    }
    if got := f.IsTest(); got != tt.want {
    t.Errorf("File.IsTest() = %v, want %v", got, tt.want)
    }
    })
    }
}

func TestFile_Content(t *testing.T) {
    content := []byte("package example")
    f := &File{
    content: content,
    }
    
    if got := f.Content(); string(got) != string(content) {
    t.Errorf("File.Content() = %v, want %v", string(got), string(content))
    }
}

func TestFile_ToPosition(t *testing.T) {
    fset := token.NewFileSet()
    // Create a file in the FileSet
    tokenFile := fset.AddFile("example.go", fset.Base(), 100)
    // Add a line at position 5
    tokenFile.AddLine(5)
    
    f := &File{
    Name: "example.go",
    Pkg: &Package{
    fset: fset,
    },
    }
    
    pos := token.Pos(tokenFile.Base() + 5)
    got := f.ToPosition(pos)
    
    if got.Filename != "example.go" {
    t.Errorf("File.ToPosition() filename = %v, want %v", got.Filename, "example.go")
    }
    if got.Offset != 5 {
    t.Errorf("File.ToPosition() offset = %v, want %v", got.Offset, 5)
    }
}

func TestFile_isMain(t *testing.T) {
    tests := []struct {
    name      string
    pkgName   string
    want      bool
    }{
    {
    name:      "main package",
    pkgName:   "main",
    want:      true,
    },
    {
    name:      "non-main package",
    pkgName:   "lint",
    want:      false,
    },
    }
    
    for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
    f := &File{
    AST: &ast.File{
        Name: &ast.Ident{
        Name: tt.pkgName,
        },
    },
    }
    if got := f.isMain(); got != tt.want {
    t.Errorf("File.isMain() = %v, want %v", got, tt.want)
    }
    })
    }
}

func TestFile_filterFailures(t *testing.T) {
    f := &File{}
    
    failures := []Failure{
    {
    RuleName: "rule1",
    Position: FailurePosition{
    Start: token.Position{Line: 10},
    End:   token.Position{Line: 10},
    },
    },
    {
    RuleName: "rule2",
    Position: FailurePosition{
    Start: token.Position{Line: 20},
    End:   token.Position{Line: 20},
    },
    },
    {
    RuleName: "rule1",
    Position: FailurePosition{
    Start: token.Position{Line: 30},
    End:   token.Position{Line: 30},
    },
    },
    }
    
    disabledIntervals := disabledIntervalsMap{
    "rule1": {
    {
    RuleName: "rule1",
    From:     token.Position{Line: 5},
    To:       token.Position{Line: 15},
    },
    },
    }
    
    // Should filter out rule1 failure at line 10 since it's in the disabled interval
    filtered := f.filterFailures(failures, disabledIntervals)
    
    if len(filtered) != 2 {
    t.Errorf("filterFailures() returned %d failures, want 2", len(filtered))
    }
    
    for _, failure := range filtered {
    if failure.RuleName == "rule1" && failure.Position.Start.Line == 10 {
    t.Errorf("filterFailures() should have filtered out rule1 failure at line 10")
    }
    }
    
    // Test with no disabled intervals
    filtered = f.filterFailures(failures, disabledIntervalsMap{})
    if len(filtered) != 3 {
    t.Errorf("filterFailures() with no disabled intervals returned %d failures, want 3", len(filtered))
    }
    
    // Test with partial overlap of disabled interval
    failures = append(failures, Failure{
    RuleName: "rule1",
    Position: FailurePosition{
    Start: token.Position{Line: 14},
    End:   token.Position{Line: 16},
    },
    })
    
    filtered = f.filterFailures(failures, disabledIntervals)
    
    for _, failure := range filtered {
    if failure.RuleName == "rule1" && 
        ((failure.Position.Start.Line >= 5 && failure.Position.Start.Line <= 15) ||
    (failure.Position.End.Line >= 5 && failure.Position.End.Line <= 15)) {
    t.Errorf("filterFailures() should have filtered out rule1 failure that overlaps with disabled interval")
    }
    }
}

func TestFile_disabledIntervals_MultipleRules(t *testing.T) {
    buildCommentGroups := func(comments ...string) []*ast.CommentGroup {
    commentGroups := make([]*ast.CommentGroup, 0, len(comments))
    for _, c := range comments {
    commentGroups = append(commentGroups, &ast.CommentGroup{
    List: []*ast.Comment{
        {Text: c},
    },
    })
    }
    return commentGroups
    }

    tests := []struct {
    name     string
    comments []*ast.CommentGroup
    expected disabledIntervalsMap
    }{
    {
    name:     "multiple rules in single directive",
    comments: buildCommentGroups("//revive:disable:rule1,rule2"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    "rule2": {
        {
        RuleName: "rule2",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    },
    },
    {
    name:     "disable with reason",
    comments: buildCommentGroups("//revive:disable:rule1 This is the reason"),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    },
    },
    {
    name:     "disable with reason requirement",
    comments: buildCommentGroups("//revive:disable:rule1"),
    expected: disabledIntervalsMap{}, // Nothing should be disabled because reason is required but not provided
    },
    {
    name:     "complex pattern with multiple enables and disables",
    comments: buildCommentGroups(
    "//revive:disable:rule1",
    "//revive:disable:rule2",
    "//revive:enable:rule1",
    "//revive:disable:rule1",
    "//revive:enable:rule2",
    ),
    expected: disabledIntervalsMap{
    "rule1": {
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        },
        },
        {
        RuleName: "rule1",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        Line:     2147483647,
        },
        },
    },
    "rule2": {
        {
        RuleName: "rule2",
        From: token.Position{
        Filename: "test.go",
        },
        To: token.Position{
        Filename: "test.go",
        },
        },
    },
    },
    },
    }

    for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
    f := &File{
    Name: "test.go",
    Pkg: &Package{
        fset: token.NewFileSet(),
    },
    AST: &ast.File{
        Comments: tt.comments,
    },
    }
    
    // Test with mustSpecifyDisableReason set to true for the third test case
    mustSpecifyReason := tt.name == "disable with reason requirement"
    got := f.disabledIntervals(nil, mustSpecifyReason, make(chan Failure, 10))
    
    if len(got) != len(tt.expected) {
    t.Errorf("disabledIntervals() = got %v items, want %v items", len(got), len(tt.expected))
    return
    }
    
    for rule, intervals := range got {
    expectedIntervals, ok := tt.expected[rule]
    if !ok {
        t.Errorf("unexpected rule %q", rule)
        continue
    }
    if len(intervals) != len(expectedIntervals) {
        t.Errorf("intervals for rule %q = got %d intervals, want %d intervals", 
        rule, len(intervals), len(expectedIntervals))
        continue
    }
    for i, interval := range intervals {
        expectedInterval := expectedIntervals[i]
        if interval.RuleName != expectedInterval.RuleName ||
        interval.From.Filename != expectedInterval.From.Filename ||
        interval.From.Line != expectedInterval.From.Line ||
        interval.To.Filename != expectedInterval.To.Filename ||
        interval.To.Line != expectedInterval.To.Line {
        t.Errorf("interval %d for rule %q = got %+v, want %+v", 
        i, rule, interval, expectedInterval)
        }
    }
    }
    })
    }
}
