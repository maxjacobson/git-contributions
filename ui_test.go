package main

import "testing"

func TestMonthLabel(t *testing.T) {
	if got := monthLabel("2026-10"); got != "October 2026" {
		t.Errorf("got %q, want %q", got, "October 2026")
	}
	if got := monthLabel("nonsense"); got != "nonsense" {
		t.Errorf("got %q, want it passed through", got)
	}
}

func TestDayLabel(t *testing.T) {
	if got := dayLabel("2024-03-15"); got != "Mar 15" {
		t.Errorf("got %q, want %q", got, "Mar 15")
	}
	if got := dayLabel("nonsense"); got != "nonsense" {
		t.Errorf("got %q, want it passed through", got)
	}
}

func TestParseActive(t *testing.T) {
	for _, yes := range []string{"y", "Y", "yes", "true", "1", " y "} {
		if !parseActive(yes) {
			t.Errorf("parseActive(%q) = false, want true", yes)
		}
	}
	for _, no := range []string{"n", "no", "false", "", "0", "maybe"} {
		if parseActive(no) {
			t.Errorf("parseActive(%q) = true, want false", no)
		}
	}
}
