package session

import (
	"regexp"
	"testing"
)

func TestGenerateRoomCodeFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z]+-\d{4}$`)

	for i := 0; i < 100; i++ {
		code := GenerateRoomCode()
		if !pattern.MatchString(code) {
			t.Fatalf("room code %q does not match expected format WORD-NNNN", code)
		}
	}
}

func TestGenerateRoomCodeUniqueness(t *testing.T) {
	codes := make(map[string]bool)
	total := 1000
	for i := 0; i < total; i++ {
		codes[GenerateRoomCode()] = true
	}

	// With ~100 words * 10000 numbers = 1M possibilities,
	// 1000 codes should have very few collisions.
	uniqueRatio := float64(len(codes)) / float64(total)
	if uniqueRatio < 0.95 {
		t.Fatalf("too many collisions: only %d unique codes out of %d (%.1f%%)",
			len(codes), total, uniqueRatio*100)
	}
}

func TestGenerateRoomCodeWordIsFromList(t *testing.T) {
	wordSet := make(map[string]bool)
	for _, w := range words {
		wordSet[w] = true
	}

	for i := 0; i < 100; i++ {
		code := GenerateRoomCode()
		// Extract word part (everything before the last dash)
		dashIdx := len(code) - 5 // "-NNNN" is 5 chars
		word := code[:dashIdx]
		if !wordSet[word] {
			t.Fatalf("word %q from code %q is not in the word list", word, code)
		}
	}
}
