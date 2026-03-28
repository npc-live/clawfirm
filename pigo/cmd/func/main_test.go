package main

import (
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary compiles the func binary into a temp directory and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "func")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func runFunc(t *testing.T, bin, formula, csvPath string) string {
	t.Helper()
	cmd := exec.Command(bin, formula, csvPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("func failed: %v\n%s", err, out)
	}
	return string(out)
}

func runFuncStdin(t *testing.T, bin, formula, input string) string {
	t.Helper()
	cmd := exec.Command(bin, formula)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("func stdin failed: %v\n%s", err, out)
	}
	return string(out)
}

func parseCSVOutput(t *testing.T, output string) (header []string, rows [][]string) {
	t.Helper()
	r := csv.NewReader(strings.NewReader(output))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse output CSV: %v", err)
	}
	if len(records) < 2 {
		t.Fatal("expected header + data rows")
	}
	return records[0], records[1:]
}

const ascCSV = `timestamp,open,high,low,close,volume
2024-01-01,100.0,105.0,98.0,103.0,1000
2024-01-02,103.0,108.0,101.0,106.0,1200
2024-01-03,106.0,110.0,104.0,109.0,1100
2024-01-04,109.0,112.0,107.0,108.0,900
2024-01-05,108.0,111.0,106.0,110.0,1300
`

const descCSV = `timestamp,open,high,low,close,volume
2024-01-05,108.0,111.0,106.0,110.0,1300
2024-01-04,109.0,112.0,107.0,108.0,900
2024-01-03,106.0,110.0,104.0,109.0,1100
2024-01-02,103.0,108.0,101.0,106.0,1200
2024-01-01,100.0,105.0,98.0,103.0,1000
`

func writeCSV(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "quote.csv")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestMA(t *testing.T) {
	bin := buildBinary(t)
	f := writeCSV(t, ascCSV)
	out := runFunc(t, bin, "ma3=ma(c,3)", f)
	header, rows := parseCSVOutput(t, out)

	// Check new column added.
	if header[len(header)-1] != "ma3" {
		t.Errorf("expected last header col 'ma3', got %q", header[len(header)-1])
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 data rows, got %d", len(rows))
	}
}

func TestSemicolonSeparator(t *testing.T) {
	bin := buildBinary(t)
	f := writeCSV(t, ascCSV)
	out := runFunc(t, bin, "ma3=ma(c,3); ema3=ema(c,3);", f)
	header, _ := parseCSVOutput(t, out)

	// Should have both ma3 and ema3 columns.
	found := map[string]bool{}
	for _, h := range header {
		found[h] = true
	}
	if !found["ma3"] || !found["ema3"] {
		t.Errorf("expected ma3 and ema3 in header, got %v", header)
	}
}

func TestDescendingOrderPreserved(t *testing.T) {
	bin := buildBinary(t)
	f := writeCSV(t, descCSV)
	out := runFunc(t, bin, "ma3=ma(c,3)", f)
	_, rows := parseCSVOutput(t, out)

	// Output should preserve descending order — first row is 2024-01-05.
	if rows[0][0] != "2024-01-05" {
		t.Errorf("expected first row timestamp 2024-01-05, got %q", rows[0][0])
	}
	if rows[len(rows)-1][0] != "2024-01-01" {
		t.Errorf("expected last row timestamp 2024-01-01, got %q", rows[len(rows)-1][0])
	}
}

func TestAscDescSameValues(t *testing.T) {
	bin := buildBinary(t)

	ascFile := writeCSV(t, ascCSV)
	descFile := writeCSV(t, descCSV)

	ascOut := runFunc(t, bin, "ma3=ma(c,3)", ascFile)
	descOut := runFunc(t, bin, "ma3=ma(c,3)", descFile)

	_, ascRows := parseCSVOutput(t, ascOut)
	_, descRows := parseCSVOutput(t, descOut)

	// Build map: timestamp → ma3 value.
	ascMap := map[string]string{}
	for _, r := range ascRows {
		ascMap[r[0]] = r[len(r)-1]
	}
	descMap := map[string]string{}
	for _, r := range descRows {
		descMap[r[0]] = r[len(r)-1]
	}

	for ts, ascVal := range ascMap {
		if descVal, ok := descMap[ts]; ok {
			if ascVal != descVal {
				t.Errorf("mismatch at %s: asc=%s desc=%s", ts, ascVal, descVal)
			}
		}
	}
}

func TestStdin(t *testing.T) {
	bin := buildBinary(t)
	out := runFuncStdin(t, bin, "ma3=ma(c,3)", ascCSV)
	header, rows := parseCSVOutput(t, out)

	if header[len(header)-1] != "ma3" {
		t.Errorf("expected last header col 'ma3', got %q", header[len(header)-1])
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 data rows, got %d", len(rows))
	}
}

func TestCaseInsensitiveColumns(t *testing.T) {
	bin := buildBinary(t)
	input := `Date,Open,High,Low,Close,Vol
2024-01-01,100,105,98,103,1000
2024-01-02,103,108,101,106,1200
2024-01-03,106,110,104,109,1100
`
	out := runFuncStdin(t, bin, "ma2=ma(c,2)", input)
	header, rows := parseCSVOutput(t, out)

	if header[len(header)-1] != "ma2" {
		t.Errorf("expected last header col 'ma2', got %q", header[len(header)-1])
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 data rows, got %d", len(rows))
	}
}

func TestNoArgs(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	err := cmd.Run()
	if err == nil {
		t.Error("expected error with no args")
	}
}
