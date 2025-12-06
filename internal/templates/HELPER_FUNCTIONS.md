# Template Helper Functions Reference

This document provides comprehensive documentation for all 60+ helper functions available in the Shiki template system.

## String Manipulation Functions

### Case Conversion

#### `upper`
Converts string to uppercase.
```go
{{upper "hello world"}} → "HELLO WORLD"
{{.FeatureName | upper}} → "USER AUTHENTICATION"
```

#### `lower`
Converts string to lowercase.
```go
{{lower "HELLO WORLD"}} → "hello world"
{{.FriendlyName | lower}} → "user authentication"
```

#### `title`
Converts string to title case (first letter of each word capitalized).
```go
{{title "hello world"}} → "Hello World"
{{.FeatureDesc | title}} → "User Authentication System"
```

### String Trimming

#### `trim`
Removes specified characters from beginning and end.
```go
{{trim "  hello world  " " "}} → "hello world"
{{trim "...data..." "."}} → "data"
```

#### `trimSpace`
Removes whitespace from beginning and end.
```go
{{trimSpace "  hello world  "}} → "hello world"
{{.UserInput | trimSpace}}
```

#### `trimPrefix`
Removes specified prefix from string.
```go
{{trimPrefix "prefix-data" "prefix-"}} → "data"
{{trimPrefix .FileName "temp_"}} → removes temp_ prefix
```

#### `trimSuffix`
Removes specified suffix from string.
```go
{{trimSuffix "data.tmp" ".tmp"}} → "data"
{{trimSuffix .FilePath ".md"}} → removes .md extension
```

### String Operations

#### `split`
Splits string into slice by delimiter.
```go
{{split "a,b,c" ","}} → ["a", "b", "c"]
{{range split .Tags ","}}
- {{.}}
{{end}}
```

#### `join`
Joins slice elements with delimiter.
```go
{{join ["a", "b", "c"] ", "}} → "a, b, c"
{{join .Dependencies ", "}}
```

#### `replace`
Replaces first occurrence of substring.
```go
{{replace "hello world" "world" "universe"}} → "hello universe"
{{replace .Description "bug" "issue"}}
```

#### `replaceAll`
Replaces all occurrences of substring.
```go
{{replaceAll "hello world world" "world" "universe"}} → "hello universe universe"
{{replaceAll .Content "\n" "<br>"}}
```

### String Testing

#### `contains`
Checks if string contains substring.
```go
{{if contains .Description "authentication"}}
This feature involves authentication.
{{end}}
```

#### `hasPrefix`
Checks if string starts with prefix.
```go
{{if hasPrefix .FileName "test_"}}
This is a test file.
{{end}}
```

#### `hasSuffix`
Checks if string ends with suffix.
```go
{{if hasSuffix .FilePath ".go"}}
This is a Go source file.
{{end}}
```

### String Formatting

#### `substring`
Extracts substring from start position with optional length.
```go
{{substring "hello world" 0 5}} → "hello"
{{substring "hello world" 6}} → "world"
{{substring .Description 0 50}}... → First 50 characters with ellipsis
```

#### `pad`
Pads string to specified length with spaces.
```go
{{pad "hello" 10}} → "hello     "
{{pad .Status 15}} → Status padded to 15 characters
```

#### `padLeft`
Pads string on the left with character.
```go
{{padLeft "42" 5 "0"}} → "00042"
{{padLeft .FeatureNumber 3 "0"}} → Zero-padded feature number
```

#### `padRight`
Pads string on the right with character.
```go
{{padRight "test" 10 "."}} → "test......"
{{padRight .Title 20 " "}} → Right-padded title
```

## Logical Operations

### Boolean Logic

#### `and`
Logical AND operation.
```go
{{if and .HasSpec .HasPlan}}
Both specification and plan exist.
{{end}}
```

#### `or`
Logical OR operation.
```go
{{if or .IsUrgent .IsBlocking}}
This requires immediate attention.
{{end}}
```

#### `not`
Logical NOT operation.
```go
{{if not .IsCompleted}}
Task is still pending.
{{end}}
```

### Conditional Operations

#### `if`
Conditional value selection (ternary-like).
```go
{{if .IsActive "Active" "Inactive"}} → Returns "Active" if true, "Inactive" if false
Status: {{if .IsEnabled "Enabled" "Disabled"}}
```

#### `default`
Returns default value if input is empty.
```go
{{default .OptionalField "No value provided"}}
{{.UserName | default "Anonymous"}}
```

#### `coalesce`
Returns first non-empty value from list.
```go
{{coalesce .NickName .FirstName .UserName "Unknown"}}
{{coalesce .CustomTitle .DefaultTitle "Untitled"}}
```

## Comparison Operations

### Basic Comparisons

#### `eq`
Tests equality.
```go
{{if eq .Status "completed"}}
Task is completed.
{{end}}
```

#### `ne`
Tests inequality.
```go
{{if ne .Priority "low"}}
High priority task.
{{end}}
```

#### `lt`
Tests less than.
```go
{{if lt .Progress 50}}
Less than 50% complete.
{{end}}
```

#### `le`
Tests less than or equal.
```go
{{if le .Attempts 3}}
Within attempt limit.
{{end}}
```

#### `gt`
Tests greater than.
```go
{{if gt .Score 85}}
Excellent score!
{{end}}
```

#### `ge`
Tests greater than or equal.
```go
{{if ge .Version 2}}
Version 2 or higher.
{{end}}
```

### Advanced Comparisons

#### `compare`
String comparison (-1, 0, 1).
```go
{{$result := compare .VersionA .VersionB}}
{{if eq $result 0}}Versions are equal{{end}}
{{if gt $result 0}}Version A is newer{{end}}
```

#### `min`
Returns minimum of numeric values.
```go
{{min .ScoreA .ScoreB .ScoreC}} → Lowest score
{{min 10 20 5}} → 5
```

#### `max`
Returns maximum of numeric values.
```go
{{max .ScoreA .ScoreB .ScoreC}} → Highest score
{{max 10 20 5}} → 20
```

## Array Operations

### Array Inspection

#### `len`
Returns length of array/slice/string.
```go
{{len .Dependencies}} → Number of dependencies
{{if gt (len .Tasks) 0}}There are {{len .Tasks}} tasks.{{end}}
```

#### `first`
Returns first element of array.
```go
{{first .Dependencies}} → First dependency
{{$primary := first .Authors}} → First author
```

#### `last`
Returns last element of array.
```go
{{last .Dependencies}} → Last dependency
{{$final := last .Versions}} → Latest version
```

#### `rest`
Returns all elements except first.
```go
{{range rest .Authors}}
Additional author: {{.}}
{{end}}
```

### Array Manipulation

#### `reverse`
Reverses array order.
```go
{{range reverse .Versions}}
Version: {{.}}
{{end}}
```

#### `sort`
Sorts array elements.
```go
{{range sort .Dependencies}}
- {{.}}
{{end}}
```

#### `arrayContains`
Checks if array contains element.
```go
{{if arrayContains "go" .Technologies}}
This project uses Go.
{{end}}
```

#### `append`
Appends elements to array.
```go
{{$newList := append .Existing "item1" "item2"}}
{{range $newList}}
- {{.}}
{{end}}
```

#### `unique`
Removes duplicate elements.
```go
{{$uniqueList := unique .Tags}}
{{join $uniqueList ", "}}
```

## File Operations

### File System

#### `fileExists`
Checks if file exists.
```go
{{if fileExists "config.json"}}
Configuration file found.
{{end}}
```

#### `readFile`
Reads file content.
```go
{{$config := readFile "settings.txt"}}
Configuration: {{$config}}
```

### Path Operations

#### `basename`
Returns filename from path.
```go
{{basename "/path/to/file.txt"}} → "file.txt"
{{basename .FilePath}} → Filename only
```

#### `dirname`
Returns directory from path.
```go
{{dirname "/path/to/file.txt"}} → "/path/to"
{{dirname .FilePath}} → Directory path
```

#### `ext`
Returns file extension.
```go
{{ext "/path/to/file.txt"}} → ".txt"
{{$extension := ext .FileName}}
```

## Utility Functions

### Time Operations

#### `now`
Returns current time.
```go
{{$current := now}}
Generated at: {{$current.Format "2006-01-02 15:04:05"}}
```

#### `formatTime`
Formats time with layout.
```go
{{formatTime .Timestamp "2006-01-02"}} → Date only
{{formatTime .CreatedAt "Jan 2, 2006"}} → Human-readable date
```

### Environment

#### `env`
Gets environment variable.
```go
{{env "USER"}} → Current username
{{$home := env "HOME"}} → Home directory
{{env "DEBUG" | default "false"}} → Debug flag with default
```

### JSON Operations

#### `toJSON`
Converts value to JSON string.
```go
{{$data := map "name" .FeatureName "id" .FeatureNumber}}
{{toJSON $data}} → {"name":"User Auth","id":42}
```

#### `fromJSON`
Parses JSON string to value.
```go
{{$config := fromJSON .ConfigString}}
Database: {{$config.database.host}}
```

### Debugging

#### `debug`
Prints debug information.
```go
{{debug .ContextData}} → Prints detailed object information
{{$value | debug}} → Debug pipeline value
```

#### `log`
Logs message (useful for debugging).
```go
{{log "Processing template for feature: " .FeatureName}}
{{$result | log "Final result:"}}
```

## Template Composition

### Template Inclusion

#### `include`
Includes another template file.
```go
{{include "shared/header.md" .}}
{{include "components/status.md" .StatusData}}
```

#### `partial`
Includes partial template with isolated context.
```go
{{partial "components/task-list.md" .TaskData}}
{{partial "shared/footer.md" .}}
```

#### `template`
Executes named template with context.
```go
{{template "task-item" .CurrentTask}}
{{template "status-badge" .}}
```

## Advanced Usage Examples

### Complex Conditionals
```go
{{if and (gt .Priority 5) (eq .Status "pending") (not .IsBlocked)}}
This is a high-priority pending task that is not blocked.
{{else if or .IsUrgent .IsOverdue}}
This task requires immediate attention.
{{else}}
Standard processing.
{{end}}
```

### Data Processing Pipeline
```go
{{$cleanTags := .Tags | unique | sort}}
{{$tagString := join $cleanTags ", "}}
{{if gt (len $cleanTags) 0}}
Tags: {{$tagString}}
{{else}}
No tags assigned.
{{end}}
```

### Dynamic Content Generation
```go
{{range $index, $task := .Tasks}}
{{$status := if $task.Completed "✓" "○"}}
{{$priority := $task.Priority | default 3}}
{{printf "%s %d. %s (P%d)" $status (add $index 1) $task.Title $priority}}
{{end}}
```

### Template Composition
```go
{{template "header" .}}

{{include "shared/navigation.md" .}}

{{if .HasContent}}
{{include "content/main.md" .}}
{{else}}
{{partial "components/empty-state.md" .}}
{{end}}

{{template "footer" .}}
```

## Function Reference Quick Guide

| Category | Functions |
|----------|-----------|
| **String Case** | `upper`, `lower`, `title` |
| **String Trim** | `trim`, `trimSpace`, `trimPrefix`, `trimSuffix` |
| **String Ops** | `split`, `join`, `replace`, `replaceAll` |
| **String Test** | `contains`, `hasPrefix`, `hasSuffix` |
| **String Format** | `substring`, `pad`, `padLeft`, `padRight` |
| **Logic** | `and`, `or`, `not`, `if`, `default`, `coalesce` |
| **Compare** | `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `compare`, `min`, `max` |
| **Array** | `len`, `first`, `last`, `rest`, `reverse`, `sort`, `arrayContains`, `append`, `unique` |
| **Files** | `fileExists`, `readFile`, `basename`, `dirname`, `ext` |
| **Time** | `now`, `formatTime` |
| **Utility** | `env`, `toJSON`, `fromJSON`, `debug`, `log` |
| **Compose** | `include`, `partial`, `template` |

All functions are designed to be pipeline-friendly and can be chained together for powerful template processing.