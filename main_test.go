package main

import (
	"os"
	"strings"
	"testing"
)

func BenchmarkBestemmia(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bestemmia()
	}
}

func TestBestemmia(t *testing.T) {
	if strings.TrimSpace(bestemmia()) == "" {
		t.Error("Generated string is empty")
	}
}

func TestGenAudio(t *testing.T) {
	_ = os.Remove("./temp/NP5M2VS4G9AQEEIPC6V6DQH5J1RGS4PE.mp3")
	uuid := genAudio("AGAGAGAGAGA")
	stat, err := os.Stat("./temp/NP5M2VS4G9AQEEIPC6V6DQH5J1RGS4PE.mp3")

	if uuid != "NP5M2VS4G9AQEEIPC6V6DQH5J1RGS4PE" {
		t.Error("Hash mismatch")
	} else {
		if err != nil {
			t.Error("File doesn't exist")
		} else {
			if !(stat.Size() > 0) {
				t.Error("File is empty")
			}
		}
	}
}
