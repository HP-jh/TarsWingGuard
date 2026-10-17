package knowledge

import (
	"testing"
)

func TestVoiceCorrectorChinese(t *testing.T) {
	vc := NewVoiceCorrector()
	tests := []struct {
		input    string
		expected string
	}{
		{"ni hao", "你好"},
		{"nihao", "你好"},
		{"xie xie", "谢谢"},
		{"xiexie", "谢谢"},
		{"qing wen", "请问"},
		{"hao de", "好的"},
		{"ming bai", "明白"},
		{"zhi dao", "知道"},
		{"bu zhidao", "不知道"},
		{"ke yi", "可以"},
		{"zai jian", "再见"},
	}
	for _, tt := range tests {
		got := vc.Correct(tt.input)
		if got != tt.expected {
			t.Errorf("Correct(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestVoiceCorrectorEnglish(t *testing.T) {
	vc := NewVoiceCorrector()
	tests := []struct {
		input    string
		expected string
	}{
		{"teh quick brown fox", "the quick brown fox"},
		{"adn ot", "and to"},
		{"taht wiht", "that with"},
		{"jsut wnat", "just want"},
		{"dont know", "don't know"},
		{"cant do it", "can't do it"},
		{"its fine", "it's fine"},
	}
	for _, tt := range tests {
		got := vc.Correct(tt.input)
		if got != tt.expected {
			t.Errorf("Correct(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestVoiceCorrectorPreserveCase(t *testing.T) {
	vc := NewVoiceCorrector()
	if got := vc.Correct("Teh"); got != "The" {
		t.Errorf("Correct(\"Teh\") = %q, want \"The\"", got)
	}
}

func TestVoiceCorrectorNoMatch(t *testing.T) {
	vc := NewVoiceCorrector()
	input := "hello world normal text"
	if got := vc.Correct(input); got != input {
		t.Errorf("Correct(%q) = %q, want unchanged", input, got)
	}
}
