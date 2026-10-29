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
