// Command func evaluates technical indicator formulas on OHLC CSV data.
//
// Usage:
//
//	func 'ma5=ma(c,5);' quote.csv >> result.csv
//	func 'ma5=ma(c,5); ema12=ema(c,12);' quote.csv
//	cat quote.csv | func 'rsi=...'
//
// The CSV must contain columns for OHLC data. Column names are detected
// automatically (case-insensitive): open, high, low, close, volume, time/timestamp/date.
// Timestamp ordering (ascending or descending) is auto-detected and data is
// always processed in chronological (ascending) order internally.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ai-gateway/pi-go/funcs"
	"github.com/elliotchance/pie/pie"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: func '<formula>' [input.csv]\n")
		fmt.Fprintf(os.Stderr, "  Example: func 'ma5=ma(c,5);' quote.csv >> result.csv\n")
		os.Exit(1)
	}

	// Strip semicolons — the parser uses whitespace-separated statements.
	formula := strings.ReplaceAll(os.Args[1], ";", " ")

	// Input: file arg or stdin.
	var reader io.Reader
	if len(os.Args) >= 3 {
		f, err := os.Open(os.Args[2])
		if err != nil {
			log.Fatalf("open %s: %v", os.Args[2], err)
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	// Parse CSV.
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("parse CSV: %v", err)
	}
	if len(records) < 2 {
		log.Fatal("CSV must have a header row and at least one data row")
	}

	header := records[0]
	data := records[1:]

	// Detect column indices (case-insensitive).
	colIdx := detectColumns(header)

	// Parse OHLCV + timestamp from rows.
	type row struct {
		open, high, low, close, volume float64
		timestamp                      string
		original                       []string
	}
	rows := make([]row, 0, len(data))
	for _, rec := range data {
		r := row{original: rec}
		if i, ok := colIdx["open"]; ok {
			r.open, _ = strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		}
		if i, ok := colIdx["high"]; ok {
			r.high, _ = strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		}
		if i, ok := colIdx["low"]; ok {
			r.low, _ = strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		}
		if i, ok := colIdx["close"]; ok {
			r.close, _ = strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		}
		if i, ok := colIdx["volume"]; ok {
			r.volume, _ = strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		}
		if i, ok := colIdx["time"]; ok {
			r.timestamp = strings.TrimSpace(rec[i])
		}
		rows = append(rows, r)
	}

	// Detect timestamp ordering and sort ascending if needed.
	reversed := false
	if len(rows) >= 2 {
		if ts, ok := colIdx["time"]; ok {
			first := strings.TrimSpace(data[0][ts])
			last := strings.TrimSpace(data[len(data)-1][ts])
			if first > last {
				reversed = true
			}
		}
	}
	if reversed {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].timestamp < rows[j].timestamp
		})
	}

	// Build OHLCV arrays.
	o := make(pie.Float64s, len(rows))
	h := make(pie.Float64s, len(rows))
	l := make(pie.Float64s, len(rows))
	c := make(pie.Float64s, len(rows))
	v := make(pie.Float64s, len(rows))
	for i, r := range rows {
		o[i] = r.open
		h[i] = r.high
		l[i] = r.low
		c[i] = r.close
		v[i] = r.volume
	}

	// Evaluate formula.
	context := map[string]interface{}{
		"o": o, "O": o, "open": o,
		"h": h, "H": h, "high": h,
		"l": l, "L": l, "low": l,
		"c": c, "C": c, "close": c,
		"v": v, "V": v, "volume": v, "vol": v,
	}
	cli := funcs.New(context)
	cli.Eval(formula)

	// Collect output variable names (formula-defined, not built-in aliases).
	builtins := map[string]bool{
		"o": true, "O": true, "open": true,
		"h": true, "H": true, "high": true,
		"l": true, "L": true, "low": true,
		"c": true, "C": true, "close": true,
		"v": true, "V": true, "volume": true, "vol": true,
	}
	var outputNames []string
	for k := range cli.Values {
		if !builtins[k] {
			outputNames = append(outputNames, k)
		}
	}
	sort.Strings(outputNames)

	if len(outputNames) == 0 {
		log.Fatal("formula produced no output variables (use 'name=expr;' syntax)")
	}

	// If reversed, reverse rows back to original order for output.
	if reversed {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
		// Also reverse each output array.
		for _, name := range outputNames {
			if arr, ok := cli.Values[name].(pie.Float64s); ok {
				for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
					arr[i], arr[j] = arr[j], arr[i]
				}
			}
		}
	}

	// Write output CSV.
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header: original columns + new indicator columns.
	outHeader := make([]string, len(header))
	copy(outHeader, header)
	outHeader = append(outHeader, outputNames...)
	writer.Write(outHeader)

	// Data rows.
	for i, r := range rows {
		outRow := make([]string, len(r.original))
		copy(outRow, r.original)
		for _, name := range outputNames {
			val := cli.Values[name]
			outRow = append(outRow, formatValue(val, i))
		}
		writer.Write(outRow)
	}
}

// detectColumns maps canonical names to column indices.
func detectColumns(header []string) map[string]int {
	m := make(map[string]int)
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "open", "o":
			m["open"] = i
		case "high", "h":
			m["high"] = i
		case "low", "l":
			m["low"] = i
		case "close", "c":
			m["close"] = i
		case "volume", "vol", "v":
			m["volume"] = i
		case "time", "timestamp", "date", "datetime", "t":
			m["time"] = i
		}
	}
	return m
}

// formatValue extracts the i-th element from an indicator result array.
func formatValue(val interface{}, i int) string {
	switch v := val.(type) {
	case pie.Float64s:
		if i < len(v) {
			return strconv.FormatFloat(v[i], 'f', 6, 64)
		}
	case []float64:
		if i < len(v) {
			return strconv.FormatFloat(v[i], 'f', 6, 64)
		}
	case float64:
		return strconv.FormatFloat(v, 'f', 6, 64)
	}
	return ""
}
