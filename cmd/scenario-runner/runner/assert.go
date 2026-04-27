package runner

import "fmt"

// evaluate checks all assertions in an AssertStep against collected events.
// lastSendMs is the scenario-start-relative timestamp of the last send step.
func evaluate(a *AssertStep, events []EventRecord, lastSendMs int64) []AssertResult {
	var results []AssertResult

	if a.GotDone != nil {
		res := AssertResult{Name: "got_done"}
		got := hasDone(events)
		if got == *a.GotDone {
			res.Passed = true
		} else if *a.GotDone {
			res.Error = "expected done event, none received"
		} else {
			res.Error = "expected no done event, but got one"
		}
		results = append(results, res)
	}

	if a.GotResponse != nil {
		res := AssertResult{Name: "got_response"}
		got := hasResponse(events)
		if got == *a.GotResponse {
			res.Passed = true
		} else if *a.GotResponse {
			res.Error = "expected non-empty response delta, none received"
		} else {
			res.Error = "expected no response, but got one"
		}
		results = append(results, res)
	}

	if a.NoError != nil {
		res := AssertResult{Name: "no_error"}
		errEv := firstError(events)
		if *a.NoError && errEv == "" {
			res.Passed = true
		} else if !*a.NoError {
			res.Passed = true
		} else {
			res.Error = "got error event: " + errEv
		}
		results = append(results, res)
	}

	if a.DoneWithin != nil {
		res := AssertResult{Name: fmt.Sprintf("done_within_%dms", *a.DoneWithin)}
		doneEv := findDone(events)
		if doneEv == nil {
			res.Error = "no done event received"
		} else {
			elapsed := doneEv.TimeMs - lastSendMs
			if int(elapsed) <= *a.DoneWithin {
				res.Passed = true
			} else {
				res.Error = fmt.Sprintf("done arrived %dms after send, limit %dms", elapsed, *a.DoneWithin)
			}
		}
		results = append(results, res)
	}

	if a.MaxGap != nil {
		res := AssertResult{Name: fmt.Sprintf("max_gap_%dms", *a.MaxGap)}
		gap := maxEventGap(events)
		if gap <= int64(*a.MaxGap) {
			res.Passed = true
		} else {
			res.Error = fmt.Sprintf("event gap %dms exceeded limit %dms", gap, *a.MaxGap)
		}
		results = append(results, res)
	}

	return results
}

func hasDone(events []EventRecord) bool {
	for _, e := range events {
		if e.Type == "done" {
			return true
		}
	}
	return false
}

func hasResponse(events []EventRecord) bool {
	for _, e := range events {
		if e.Type == "delta" && e.Content != "" {
			return true
		}
	}
	return false
}

func firstError(events []EventRecord) string {
	for _, e := range events {
		if e.Type == "error" {
			return e.Content
		}
	}
	return ""
}

func findDone(events []EventRecord) *EventRecord {
	for i := range events {
		if events[i].Type == "done" {
			return &events[i]
		}
	}
	return nil
}

func maxEventGap(events []EventRecord) int64 {
	var max int64
	for i := 1; i < len(events); i++ {
		gap := events[i].TimeMs - events[i-1].TimeMs
		if gap > max {
			max = gap
		}
	}
	return max
}
