package design

import (
	"testing"
)

func TestBuildStyles_CoreStylesRender(t *testing.T) {
	ts := DefaultTokens()
	ss := BuildStyles(ts)

	// Verify core styles produce output (non-panic, non-empty).
	if ss.User.Render("test") == "" {
		t.Error("User style should render non-empty")
	}
	if ss.Assistant.Render("test") == "" {
		t.Error("Assistant style should render non-empty")
	}
	if ss.Error.Render("test") == "" {
		t.Error("Error style should render non-empty")
	}
}

func TestBuildStyles_AllFieldsRender(t *testing.T) {
	ts := DefaultTokens()
	ss := BuildStyles(ts)

	// Spot-check that all style categories produce output.
	styles := map[string]string{
		"ToolName":    ss.ToolName.Render("x"),
		"ToolSuccess": ss.ToolSuccess.Render("x"),
		"DiffAdd":     ss.DiffAdd.Render("x"),
		"DiffDel":     ss.DiffDel.Render("x"),
		"HealthGreen": ss.HealthGreen.Render("x"),
		"HealthRed":   ss.HealthRed.Render("x"),
		"BudgetOK":    ss.BudgetOK.Render("x"),
		"ZoneCold":    ss.ZoneCold.Render("x"),
		"DriftGood":   ss.DriftGood.Render("x"),
		"ModeInsert":  ss.ModeInsert.Render("x"),
		"SepFocus":    ss.SepFocus.Render("x"),
	}
	for name, rendered := range styles {
		if rendered == "" {
			t.Errorf("%s style should render non-empty", name)
		}
	}
}

func TestActiveStyles_InitializedAtStartup(t *testing.T) {
	// ActiveStyles is set via init() — should produce output.
	if ActiveStyles.User.Render("test") == "" {
		t.Error("ActiveStyles.User should be initialized")
	}
}

func TestBuildStyles_DifferentThemes(t *testing.T) {
	defaultSS := BuildStyles(TokensFromTheme(DefaultTheme))
	claudeSS := BuildStyles(TokensFromTheme(ClaudeTheme))

	// Different themes should produce output without panic.
	_ = defaultSS.User.Render("test")
	_ = claudeSS.User.Render("test")
}
