package main

import (
	"testing"
)

func TestTrieMask(t *testing.T) {
	secrets := []string{"secret", "password"}
	trie := BuildTrieFromSecrets(secrets)

	tests := []struct {
		input    string
		expected string
	}{
		{"this is a secret", "this is a ******"},
		{"my password is strong", "my ******** is strong"},
		{"should not be masked", "should not be masked"},
		{"partial match secre", "partial match *****"},
	}

	for _, test := range tests {
		masked, remaining, state := trie.Mask(test.input, NewTrieNodeState())
		masked += trie.PrintRemaining(remaining, state)

		if masked != test.expected {
			t.Errorf("expected %q, got %q", test.expected, masked)
		}
	}
}

func TestTrieMaskDifficult(t *testing.T) {
	secrets := []string{"secret", "secrets and password"}
	trie := BuildTrieFromSecrets(secrets)

	tests := []struct {
		input    string
		expected string
	}{
		{"this is a secret", "this is a ******"},
		{"my password is strong", "my password is strong"},
		{"multiple secrets and password", "multiple ********************"},
		{"multiple secrets and passwor!", "multiple ******s and passwor!"},
	}

	for _, test := range tests {
		masked, remaining, state := trie.Mask(test.input, NewTrieNodeState())
		masked += trie.PrintRemaining(remaining, state)

		if masked != test.expected {
			t.Errorf("expected %q, got %q", test.expected, masked)
		}
	}
}

func TestTrieMaskWithIntermediateState(t *testing.T) {
	secrets := []string{"secret", "password", "token"}
	trie := BuildTrieFromSecrets(secrets)

	tests := []struct {
		inputs   []string
		expected string
	}{
		{[]string{"this is a sec", "ret"}, "this is a ******"},
		{[]string{"my pass", "word is strong"}, "my ******** is strong"},
		{[]string{"use this to", "ken"}, "use this *****"},
		{[]string{"partial match sec", "re"}, "partial match *****"},
		{[]string{"multiple sec", "ret and pass", "word"}, "multiple ****** and ********"},
	}

	for _, test := range tests {
		currentState := NewTrieNodeState()
		remaining := ""
		var result string
		for _, input := range test.inputs {
			var masked string
			remaining += input
			masked, remaining, currentState = trie.Mask(remaining, currentState)
			result += masked
		}
		result += trie.PrintRemaining(remaining, currentState)

		if result != test.expected {
			t.Errorf("expected %q, got %q", test.expected, result)
		}
	}
}

func TestTrieMaskWithBackslash(t *testing.T) {
	secrets := []string{"\"secret\""}
	trie := BuildTrieFromSecrets(secrets)

	tests := []struct {
		input    string
		expected string
	}{
		{"this is not a secret", "this is not a secret"},
		{"this is a \"secret\"", "this is a ********"},
		{"this is also a \\\"secret\"", "this is also a *********"},
		{"this is also a \\\"secret\\\"", "this is also a **********"},
	}

	for _, test := range tests {
		masked, remaining, state := trie.Mask(test.input, NewTrieNodeState())
		masked += trie.PrintRemaining(remaining, state)

		if masked != test.expected {
			t.Errorf("expected %q, got %q", test.expected, masked)
		}
	}
}
