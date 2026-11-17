package main

import (
	"testing"

	"github.com/Crandel/gmail/internal/accounts"
)

func TestOutputFormat(t *testing.T) {
	t.Parallel()
	accs := []accounts.Account{
		{Short: "A", MailType: accounts.Gmail},
		{Short: "B", MailType: accounts.Gmail},
	}
	counts := []int64{3, 5}

	segments := make([]string, len(accs))
	for i, acc := range accs {
		segments[i] = accountSegment(acc, counts[i])
	}

	got := formatOutput(segments)
	want := "A:3B:5\n"
	if got != want {
		t.Errorf("formatOutput() = %q, want %q", got, want)
	}
}
