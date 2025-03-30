package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jonathanhecl/subtitle-processor/subtitles"
	modelSubtitles "github.com/jonathanhecl/subtitle-processor/subtitles/models"
)

func TestBuildTranslationPrompt(t *testing.T) {
	// Test case 1: Without context
	lines := []modelSubtitles.ModelItemSubtitle{
		{Text: []string{"Hello, how are you?"}},
		{Text: []string{"I'm fine, thank you."}},
	}
	context := []string{}

	prompt := buildTranslationPrompt(lines, context)

	// Check that the prompt contains the instructions for Spanish
	if !strings.Contains(prompt, "Español neutro") {
		t.Errorf("Prompt should specify neutral Spanish translation")
	}

	// Check that the prompt contains the lines to translate
	if !strings.Contains(prompt, "Hello, how are you?") {
		t.Errorf("Prompt should contain the lines to translate")
	}

	// Test case 2: With context
	context = []string{"Hola, ¿cómo estás?"}
	prompt = buildTranslationPrompt(lines, context)

	// Check that the prompt contains the context
	if !strings.Contains(prompt, "Previous successfully translated lines in Español neutro") || !strings.Contains(prompt, "Hola, ¿cómo estás?") {
		t.Errorf("Prompt should contain the context")
	}
}

func TestProcessTranslatedResponse(t *testing.T) {
	// Create duration objects for timing
	oneSecond, _ := time.ParseDuration("1s")
	twoSecond, _ := time.ParseDuration("2s")
	threeSecond, _ := time.ParseDuration("3s")
	fourSecond, _ := time.ParseDuration("4s")

	// Original lines
	originalLines := []modelSubtitles.ModelItemSubtitle{
		{Seq: 1, Start: oneSecond, End: twoSecond, Text: []string{"Hello"}},
		{Seq: 2, Start: threeSecond, End: fourSecond, Text: []string{"World"}},
	}

	// Test case 1: Response with same number of lines
	response := "Hola\nMundo"
	translatedLines := processTranslatedResponse(response, originalLines)

	if len(translatedLines) != 2 {
		t.Errorf("Expected 2 translated lines, got %d", len(translatedLines))
	}

	// Check that the first line has both translated texts
	if len(translatedLines[0].Text) != 2 {
		t.Errorf("Expected 2 text items in first line, got %d", len(translatedLines[0].Text))
	}
	
	if translatedLines[0].Text[0] != "Hola" || translatedLines[0].Text[1] != "Mundo" {
		t.Errorf("Incorrect translation in first line: %v", translatedLines[0].Text)
	}

	// Check that timing information is preserved
	if translatedLines[0].Start != oneSecond || translatedLines[0].End != twoSecond {
		t.Errorf("Timing information not preserved: %v", translatedLines[0])
	}

	// Test case 2: Response with fewer lines
	response = "Hola"
	translatedLines = processTranslatedResponse(response, originalLines)

	if len(translatedLines) != 2 {
		t.Errorf("Expected 2 translated lines, got %d", len(translatedLines))
	}

	// Check that the first line has the translated text
	if len(translatedLines[0].Text) != 1 {
		t.Errorf("Expected 1 text item in first line, got %d", len(translatedLines[0].Text))
	}
	
	if translatedLines[0].Text[0] != "Hola" {
		t.Errorf("First line should be translated: %v", translatedLines[0])
	}

	// Second line should be preserved from original
	if translatedLines[1].Text[0] != "World" {
		t.Errorf("Second line should be preserved from original: %v", translatedLines[1])
	}

	// Test case 3: Response with extra lines
	response = "Hola\nMundo\nExtra"
	translatedLines = processTranslatedResponse(response, originalLines)

	if len(translatedLines) != 2 {
		t.Errorf("Expected 2 translated lines, got %d", len(translatedLines))
	}

	// Check that the first line has all three translated texts
	if len(translatedLines[0].Text) != 3 {
		t.Errorf("Expected 3 text items in first line, got %d", len(translatedLines[0].Text))
	}
	
	if translatedLines[0].Text[0] != "Hola" || 
	   translatedLines[0].Text[1] != "Mundo" || 
	   translatedLines[0].Text[2] != "Extra" {
		t.Errorf("Incorrect translation in first line: %v", translatedLines[0].Text)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test min function
	if min(5, 10) != 5 {
		t.Errorf("min(5, 10) should be 5")
	}
	if min(10, 5) != 5 {
		t.Errorf("min(10, 5) should be 5")
	}

	// Test max function
	if max(5, 10) != 10 {
		t.Errorf("max(5, 10) should be 10")
	}
	if max(10, 5) != 10 {
		t.Errorf("max(10, 5) should be 10")
	}
}

// Integration test with mock data
func TestIntegrationWithMockData(t *testing.T) {
	// Create a temporary subtitle file for testing
	tempFile, err := os.CreateTemp("", "test-subtitle-*.srt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test subtitle content
	testContent := `1
00:00:01,000 --> 00:00:02,000
Hello

2
00:00:03,000 --> 00:00:04,000
World
`
	if _, err := tempFile.Write([]byte(testContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Create a mock translation function for testing
	mockTranslate := func(ctx context.Context, prompt string) (string, error) {
		// Simple mock that returns Spanish translations
		if strings.Contains(prompt, "Hello") {
			return "Hola", nil
		}
		if strings.Contains(prompt, "World") {
			return "Mundo", nil
		}
		return "", nil
	}

	// Load the test subtitle file
	sub := subtitles.Subtitle{}
	err = sub.LoadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load subtitle file: %v", err)
	}

	// Create target subtitle structure
	translate := subtitles.Subtitle{}
	translate.Filename = tempFile.Name() + ".translated.srt"
	translate.Format = sub.Format
	translate.Lines = make([]modelSubtitles.ModelItemSubtitle, len(sub.Lines))

	// Process each line manually for testing
	for i, line := range sub.Lines {
		// Mock translation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Build a simple prompt
		prompt := "Translate: " + strings.Join(line.Text, " ")

		// Get mock translation
		translatedText, err := mockTranslate(ctx, prompt)
		if err != nil {
			t.Fatalf("Mock translation failed: %v", err)
		}

		// Create a new line with the translated text
		newLine := modelSubtitles.ModelItemSubtitle{
			Seq:   line.Seq,
			Start: line.Start,
			End:   line.End,
			Text:  []string{strings.TrimSpace(translatedText)},
		}
		translate.Lines[i] = newLine
	}

	// Verify translations
	if len(translate.Lines) != 2 {
		t.Errorf("Expected 2 translated lines, got %d", len(translate.Lines))
	}

	if translate.Lines[0].Text[0] != "Hola" {
		t.Errorf("First line should be 'Hola', got '%s'", translate.Lines[0].Text[0])
	}

	if translate.Lines[1].Text[0] != "Mundo" {
		t.Errorf("Second line should be 'Mundo', got '%s'", translate.Lines[1].Text[0])
	}
}
