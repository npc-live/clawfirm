package whipflow

import "testing"

func TestAnalyzeComplexity_Simple(t *testing.T) {
	src := `session "Hello world"`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierSimple {
		t.Errorf("tier = %q, want %q", a.Tier, TierSimple)
	}
	if a.SessionCount != 1 {
		t.Errorf("session_count = %d, want 1", a.SessionCount)
	}
	if a.HasAsk || a.HasParallel || a.HasLoops || a.HasChoice {
		t.Error("expected no flags set for simple program")
	}
	if len(a.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(a.Steps))
	}
	if a.Steps[0].Prompt != "Hello world" {
		t.Errorf("step prompt = %q, want %q", a.Steps[0].Prompt, "Hello world")
	}
}

func TestAnalyzeComplexity_SimpleLetBinding(t *testing.T) {
	src := `let result = session "What is Go?"`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierSimple {
		t.Errorf("tier = %q, want %q", a.Tier, TierSimple)
	}
	if a.SessionCount != 1 {
		t.Errorf("session_count = %d, want 1", a.SessionCount)
	}
	if len(a.Steps) != 1 || a.Steps[0].Name != "result" {
		t.Errorf("expected step named 'result', got %v", a.Steps)
	}
}

func TestAnalyzeComplexity_Medium(t *testing.T) {
	src := `
let step1 = session "Research topic"
let step2 = session "Summarize findings"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierMedium {
		t.Errorf("tier = %q, want %q", a.Tier, TierMedium)
	}
	if a.SessionCount != 2 {
		t.Errorf("session_count = %d, want 2", a.SessionCount)
	}
	if len(a.Steps) != 2 {
		t.Errorf("steps = %d, want 2", len(a.Steps))
	}
}

func TestAnalyzeComplexity_MediumFourSessions(t *testing.T) {
	src := `
let a = session "Step 1"
let b = session "Step 2"
let c = session "Step 3"
let d = session "Step 4"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierMedium {
		t.Errorf("tier = %q, want %q", a.Tier, TierMedium)
	}
	if a.SessionCount != 4 {
		t.Errorf("session_count = %d, want 4", a.SessionCount)
	}
}

func TestAnalyzeComplexity_ComplexFiveSessions(t *testing.T) {
	src := `
let a = session "Step 1"
let b = session "Step 2"
let c = session "Step 3"
let d = session "Step 4"
let e = session "Step 5"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierComplex {
		t.Errorf("tier = %q, want %q", a.Tier, TierComplex)
	}
}

func TestAnalyzeComplexity_AskForcesPreview(t *testing.T) {
	src := `
ask topic: "What topic?"
let result = session "Research {topic}"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if !a.HasAsk {
		t.Error("expected HasAsk=true")
	}
	// 1 session + ask → not simple (falls to complex since not 2-4 sessions)
	if a.Tier == TierSimple {
		t.Error("ask should prevent simple tier")
	}
}

func TestAnalyzeComplexity_Parallel(t *testing.T) {
	src := `
parallel:
  session "Task A"
  session "Task B"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if !a.HasParallel {
		t.Error("expected HasParallel=true")
	}
	if a.Tier != TierComplex {
		t.Errorf("tier = %q, want %q", a.Tier, TierComplex)
	}
	if a.SessionCount != 2 {
		t.Errorf("session_count = %d, want 2", a.SessionCount)
	}
}

func TestAnalyzeComplexity_LoopComplex(t *testing.T) {
	src := `
repeat 3:
  session "Iteration"
`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if !a.HasLoops {
		t.Error("expected HasLoops=true")
	}
	if a.Tier != TierComplex {
		t.Errorf("tier = %q, want %q", a.Tier, TierComplex)
	}
}

func TestAnalyzeComplexity_PromptTruncation(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	src := `session "` + long + `"`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if len(a.Steps) != 1 {
		t.Fatal("expected 1 step")
	}
	if len(a.Steps[0].Prompt) > 124 { // 120 + "..."
		t.Errorf("prompt not truncated: len=%d", len(a.Steps[0].Prompt))
	}
}

func TestAnalyzeComplexity_EmptyProgram(t *testing.T) {
	src := `# just a comment`
	prog, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	a := AnalyzeComplexity(prog)
	if a.Tier != TierSimple {
		t.Errorf("tier = %q, want %q", a.Tier, TierSimple)
	}
	if a.SessionCount != 0 {
		t.Errorf("session_count = %d, want 0", a.SessionCount)
	}
}
