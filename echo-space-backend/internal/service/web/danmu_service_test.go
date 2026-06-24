package web

import (
	"strings"
	"testing"
)

func TestValidateLoadDanmuInput(t *testing.T) {
	valid := LoadDanmuInput{
		FileID:  "Abc123Def456Ghi78901",
		VideoID: "Abc123Def4",
	}
	if err := validateLoadDanmuInput(valid); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*LoadDanmuInput)
	}{
		{name: "invalid video id", mutate: func(input *LoadDanmuInput) { input.VideoID = "bad" }},
		{name: "invalid file id", mutate: func(input *LoadDanmuInput) { input.FileID = "bad" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validateLoadDanmuInput(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidatePostDanmuInput(t *testing.T) {
	valid := PostDanmuInput{
		UserID:  "10001",
		Text:    "test danmu",
		Mode:    1,
		Color:   "#ffffff",
		Time:    10,
		FileID:  "Abc123Def456Ghi78901",
		VideoID: "Abc123Def4",
	}
	if err := validatePostDanmuInput(valid); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*PostDanmuInput)
	}{
		{name: "empty text", mutate: func(input *PostDanmuInput) { input.Text = "" }},
		{name: "long text", mutate: func(input *PostDanmuInput) { input.Text = strings.Repeat("x", 201) }},
		{name: "invalid mode", mutate: func(input *PostDanmuInput) { input.Mode = -1 }},
		{name: "invalid time", mutate: func(input *PostDanmuInput) { input.Time = -1 }},
		{name: "invalid video id", mutate: func(input *PostDanmuInput) { input.VideoID = "bad" }},
		{name: "invalid file id", mutate: func(input *PostDanmuInput) { input.FileID = "bad" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if err := validatePostDanmuInput(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
