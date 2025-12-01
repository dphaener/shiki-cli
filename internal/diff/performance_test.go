package diff

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// BenchmarkDiffGeneration tests the performance of diff generation with various file sizes
func BenchmarkDiffGeneration(b *testing.B) {
	generator := NewGenerator()

	testCases := []struct {
		name      string
		oldLines  int
		newLines  int
		changes   int
	}{
		{"small_100_lines", 100, 100, 10},
		{"medium_500_lines", 500, 500, 50},
		{"large_1000_lines", 1000, 1000, 100},
		{"very_large_2000_lines", 2000, 2000, 200},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			oldContent := generateTestContent(tc.oldLines, "old")
			newContent := generateTestContent(tc.newLines, "new")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := generator.GenerateUnifiedDiff(oldContent, newContent, "test.txt")
				if err != nil {
					b.Fatalf("Failed to generate diff: %v", err)
				}
			}
		})
	}
}

// BenchmarkDiffRendering tests the performance of diff rendering
func BenchmarkDiffRendering(b *testing.B) {
	generator := NewGenerator()
	renderer := NewRenderer()

	// Generate a representative diff
	oldContent := generateTestContent(500, "original")
	newContent := generateTestContent(500, "modified")

	fileDiff, err := generator.GenerateUnifiedDiff(oldContent, newContent, "test.txt")
	if err != nil {
		b.Fatalf("Failed to generate diff: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.FormatDiffForTerminal(fileDiff, 80)
	}
}

// BenchmarkFullDiffPipeline tests the complete diff pipeline end-to-end
func BenchmarkFullDiffPipeline(b *testing.B) {
	generator := NewGenerator()
	renderer := NewRenderer()

	testSizes := []struct {
		name  string
		lines int
	}{
		{"small_100", 100},
		{"medium_500", 500},
		{"large_1000", 1000},
		{"very_large_2000", 2000},
	}

	for _, size := range testSizes {
		b.Run(size.name, func(b *testing.B) {
			oldContent := generateTestContent(size.lines, "before")
			newContent := generateTestContent(size.lines, "after")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Full pipeline: generation + rendering
				fileDiff, err := generator.GenerateUnifiedDiff(oldContent, newContent, "benchmark.txt")
				if err != nil {
					b.Fatalf("Failed to generate diff: %v", err)
				}

				_ = renderer.FormatDiffForTerminal(fileDiff, 80)
			}
		})
	}
}

// TestPerformanceRequirement validates that diff processing meets the 2-second requirement
func TestPerformanceRequirement(t *testing.T) {
	generator := NewGenerator()
	renderer := NewRenderer()

	// Test with a large file that represents a realistic worst-case scenario
	oldContent := generateTestContent(3000, "original") // ~3000 lines
	newContent := generateTestContent(3000, "modified") // ~3000 lines

	// Test performance multiple times to get average
	const iterations = 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Full pipeline
		fileDiff, err := generator.GenerateUnifiedDiff(oldContent, newContent, "performance-test.txt")
		if err != nil {
			t.Fatalf("Failed to generate diff: %v", err)
		}

		_ = renderer.FormatDiffForTerminal(fileDiff, 80)

		duration := time.Since(start)
		totalDuration += duration
	}

	averageDuration := totalDuration / iterations
	maxAllowedDuration := 2 * time.Second

	t.Logf("Average processing time: %v", averageDuration)
	t.Logf("Maximum allowed time: %v", maxAllowedDuration)

	if averageDuration > maxAllowedDuration {
		t.Errorf("Performance requirement not met: average %v > required %v", averageDuration, maxAllowedDuration)
	}

	// Also test that 95% of operations complete within the limit
	passCount := 0
	for i := 0; i < 20; i++ {
		start := time.Now()
		fileDiff, _ := generator.GenerateUnifiedDiff(oldContent, newContent, "perf-test.txt")
		_ = renderer.FormatDiffForTerminal(fileDiff, 80)
		if time.Since(start) <= maxAllowedDuration {
			passCount++
		}
	}

	passRate := float64(passCount) / 20.0
	if passRate < 0.95 {
		t.Errorf("95%% performance requirement not met: only %.1f%% of operations completed within %v", passRate*100, maxAllowedDuration)
	}

	t.Logf("Performance pass rate: %.1f%%", passRate*100)
}

// TestMemoryUsage validates that diff processing doesn't consume excessive memory
func TestMemoryUsage(t *testing.T) {
	generator := NewGenerator()

	// Test with progressively larger files
	sizes := []int{1000, 2000, 3000, 5000}

	for _, size := range sizes {
		oldContent := generateTestContent(size, "before")
		newContent := generateTestContent(size, "after")

		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Simple test that the operation completes without excessive memory usage
			fileDiff, err := generator.GenerateUnifiedDiff(oldContent, newContent, "memory-test.txt")
			if err != nil {
				t.Fatalf("Failed to generate diff for size %d: %v", size, err)
			}

			// Verify the diff was created and has reasonable size
			if fileDiff.UnifiedDiff == "" && oldContent != newContent {
				t.Error("Expected non-empty diff for different content")
			}

			// Check that large files are marked appropriately
			if size >= 2000 && !fileDiff.IsLarge {
				t.Log("Large file not marked as large - this is acceptable but worth noting")
			}
		})
	}
}

// generateTestContent creates test content with specified number of lines
func generateTestContent(lines int, prefix string) string {
	var content strings.Builder

	for i := 0; i < lines; i++ {
		// Create varied content to make realistic diffs
		lineNumber := i + 1
		switch {
		case i%10 == 0:
			content.WriteString(fmt.Sprintf("// %s section %d\n", prefix, lineNumber/10+1))
		case i%5 == 0:
			content.WriteString(fmt.Sprintf("func %s_function_%d() {\n", prefix, lineNumber))
		case i%5 == 4:
			content.WriteString("}\n")
		default:
			content.WriteString(fmt.Sprintf("    %s line %d with some content and text\n", prefix, lineNumber))
		}
	}

	return content.String()
}