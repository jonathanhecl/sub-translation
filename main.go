package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jonathanhecl/gollama"
	"github.com/jonathanhecl/gotimeleft"
	"github.com/jonathanhecl/subtitle-processor/subtitles"
	modelSubtitles "github.com/jonathanhecl/subtitle-processor/subtitles/models"
)

var (
	version      = "1.0.0"
	source       = ""
	target       = ""
	model        = "phi4"
	originalLang = "English"
	lang         = "Español neutro"
	// Variables para seguimiento del progreso
	progress      *gotimeleft.TimeLeft
	progressMutex sync.Mutex
)

// TranslationJob represents a batch of subtitle lines to translate
type TranslationJob struct {
	startIndex int
	lines      []modelSubtitles.ModelItemSubtitle
	context    []string
}

// TranslationResult represents the result of a translation job
type TranslationResult struct {
	startIndex int
	lines      []modelSubtitles.ModelItemSubtitle
	err        error
}

func main() {
	fmt.Println("Sub-Translation v" + version)

	// Parse command line arguments
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-s=") {
			source = strings.TrimPrefix(arg, "-s=")
		} else if strings.HasPrefix(arg, "-t=") {
			target = strings.TrimPrefix(arg, "-t=")
		} else if strings.HasPrefix(arg, "-o=") {
			originalLang = strings.TrimPrefix(arg, "-o=")
		} else if strings.HasPrefix(arg, "-l=") {
			lang = strings.TrimPrefix(arg, "-l=")
		} else if strings.HasPrefix(arg, "-m=") {
			model = strings.TrimPrefix(arg, "-m=")
		}
	}

	if source == "" {
		fmt.Print("Usage: sub-translation -s=<source.srt> [-t=<target.srt>] [-o=<language>] [-l=<language>] [-m=<model>]\n")
		return
	}

	if _, err := os.Stat(source); os.IsNotExist(err) {
		fmt.Printf("Source file %s does not exist\n", source)
		return
	}

	if originalLang == "" {
		originalLang = "English"
	}

	if target == "" {
		// Auto-generate target filename
		target = strings.TrimSuffix(source, ".srt") + "_translated.srt"
	}

	fmt.Println("Source:", source)
	fmt.Println("Target:", target)
	fmt.Println("Original Language:", originalLang)
	fmt.Println("Language:", lang)
	fmt.Println("Model:", model)
	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Initialize Gollama
	g := gollama.New(model)
	// g.Verbose = true
	if err := g.PullIfMissing(ctx); err != nil {
		fmt.Printf("Error initializing model: %v\n", err)
		return
	}

	// Load source subtitle file
	sub := subtitles.Subtitle{}
	err := sub.LoadFile(source)
	if err != nil {
		fmt.Printf("Error loading subtitle file: %v\n", err)
		return
	}

	if len(sub.Lines) < 1 {
		fmt.Println("Not enough lines in subtitle file")
		return
	}

	// Initialize counters
	progress = gotimeleft.Init(len(sub.Lines))
	progress.Value(0)
	fmt.Printf("Total lines to translate: %d\n\n", len(sub.Lines))
	fmt.Println("Starting translation...")

	// Initialize target subtitle structure
	translate := subtitles.Subtitle{}
	translate.Filename = target
	translate.Format = sub.Format
	translate.Lines = make([]modelSubtitles.ModelItemSubtitle, len(sub.Lines))

	// Start progress display
	stopProgress := make(chan struct{})
	go showProgress(stopProgress)

	// Process subtitles sequentially
	processSubtitles(ctx, g, sub, &translate)

	// Detener la goroutine de progreso
	close(stopProgress)

	// Mostrar progreso final
	fmt.Printf("\nTranslation progress: %s %s lines (100%%) - Total time: %s\n", progress.GetProgressBar(64), progress.GetProgressValues(), progress.GetTimeSpent().String())

	// Save translated subtitles
	fmt.Println("Saving translated subtitles to", translate.Filename)
	if err := translate.SaveFile(translate.Filename); err != nil {
		fmt.Printf("Error saving translated subtitles: %v\n", err)
		return
	}

	fmt.Printf("\nTranslation completed successfully!\n")
}

// showProgress shows progress every 2 seconds
func showProgress(stop chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progressMutex.Lock()
			fmt.Printf("\rTranslation progress: %s %s lines (%s) - Total time: %s - Left time: %s\n", progress.GetProgressBar(64), progress.GetProgressValues(), progress.GetProgress(1), progress.GetTimeSpent().String(), progress.GetTimeLeft().String())
			progressMutex.Unlock()
		case <-stop:
			return
		}
	}
}

// processSubtitles processes subtitles sequentially with sliding context window
func processSubtitles(ctx context.Context, g *gollama.Gollama, source subtitles.Subtitle, target *subtitles.Subtitle) {
	// Process all lines sequentially with sliding context window
	for i := 0; i < len(source.Lines); i++ {
		// Build context based on the line number
		var context []string

		// Special handling for first three lines:
		// - Line 1: No context
		// - Line 2: Just line 1 as context
		// - Line 3: Lines 1 and 2 as context
		// - Line 4+: Previous three lines as context
		if i > 0 {
			// Calculate how many previous lines to include as context
			contextSize := min(i, 3)
			startIdx := i - contextSize

			// Add previous translated lines as context
			for j := startIdx; j < i; j++ {
				if len(target.Lines[j].Text) > 0 {
					context = append(context, strings.Join(target.Lines[j].Text, "\n"))
				}
			}
		}

		// fmt.Printf("Translating line %d\n", i+1)

		// Try main translation approach first with the appropriate context
		translated, err := attemptTranslation(ctx, g, []modelSubtitles.ModelItemSubtitle{source.Lines[i]}, context, -1)

		// If main approach failed, try alternative strategies
		if err != nil || len(translated) == 0 {
			fmt.Printf("First attempt failed for line %d, trying alternative strategies\n", i+1)
			var success bool

			// Try each alternative strategy
			for strategy := 1; strategy <= 3 && !success; strategy++ {
				fmt.Printf("Trying alternative strategy %d for line %d\n", strategy, i+1)
				translated, err = attemptTranslation(ctx, g, []modelSubtitles.ModelItemSubtitle{source.Lines[i]}, nil, strategy)
				if err == nil && len(translated) > 0 {
					success = true
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			// If all strategies failed, use original line
			if !success {
				fmt.Printf("All translation attempts failed for line %d, using original\n", i+1)
				translated = []modelSubtitles.ModelItemSubtitle{source.Lines[i]}
			}
		}

		// Update target with translated line
		if len(translated) > 0 {
			target.Lines[i] = translated[0]
		} else {
			target.Lines[i] = source.Lines[i]
		}

		// Update progress counter
		progressMutex.Lock()
		progress.Step(1)
		progressMutex.Unlock()
	}
}

func attemptTranslation(ctx context.Context, g *gollama.Gollama, lines []modelSubtitles.ModelItemSubtitle, contextLines []string, strategy int) ([]modelSubtitles.ModelItemSubtitle, error) {
	// Create a context with timeout for this specific translation
	translationCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Build the translation prompt based on strategy
	var prompt string
	if strategy < 0 {
		prompt = buildTranslationPrompt(lines, contextLines)
	} else {
		prompt = buildAlternativePrompt(strategy, lines)
	}

	type outputType struct {
		Translation string `description:"Translation"`
	}

	// Get translation from Gollama
	response, err := g.Chat(translationCtx, prompt, gollama.StructToStructuredFormat(outputType{}))
	if err != nil {
		return nil, err
	}

	// Check if response is empty
	var output outputType
	response.DecodeContent(&output)

	// fmt.Println("Input prompt:")
	// fmt.Println(prompt)
	// fmt.Println("Translation response:", output.Translation)

	if strings.TrimSpace(output.Translation) == "" {
		return nil, fmt.Errorf("empty translation response")
	}

	// Process translated lines
	translatedLines := processTranslatedResponse(output.Translation, lines)

	// Check if we got any translations
	if len(translatedLines) == 0 {
		return nil, fmt.Errorf("no valid translations in response")
	}

	return translatedLines, nil
}

func buildTranslationPrompt(linesToTranslate []modelSubtitles.ModelItemSubtitle, contextLines []string) string {
	var prompt strings.Builder

	// Add context if available
	if len(contextLines) > 0 {
		prompt.WriteString(fmt.Sprintf("Previous successfully translated lines in %s (for context):\n", lang))
		for i, ctx := range contextLines {
			prompt.WriteString(fmt.Sprintf("Line %d: %s\n", i+1, ctx))
		}
		prompt.WriteString("\n")
	}

	// Add the translation instruction with more emphasis on maintaining coherence with context
	if len(contextLines) > 0 {
		prompt.WriteString(fmt.Sprintf("Based on the above context, translate the following subtitle line from %s to %s.\n", originalLang, lang))
		prompt.WriteString("Maintain consistency with previous translations, same style and tone, ensuring the translation flows naturally.\n\n")
	} else {
		prompt.WriteString(fmt.Sprintf("Translate the following subtitle line from %s to %s.\n", originalLang, lang))
		prompt.WriteString("Use natural and fluent language while maintaining the original style and tone.\n\n")
	}

	// Add the line to translate (usually just one line at a time)
	prompt.WriteString(fmt.Sprintf("Line to translate: %s\n", strings.Join(linesToTranslate[0].Text, " ")))

	return prompt.String()
}

// Alternative prompt strategies to try if the first one fails
func buildAlternativePrompt(strategy int, linesToTranslate []modelSubtitles.ModelItemSubtitle) string {
	var prompt strings.Builder

	switch strategy {
	case 1:
		// Strategy 1: More direct and simple approach
		prompt.WriteString(fmt.Sprintf("Translate each of these lines from %s to %s:\n\n", originalLang, lang))
		for i, line := range linesToTranslate {
			prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.Join(line.Text, " ")))
		}
	case 2:
		// Strategy 2: Request line-by-line format in the response
		prompt.WriteString(fmt.Sprintf("Translate the following %s text to %s. Return ONLY the translated text, one subtitle per line:\n\n", originalLang, lang))
		for _, line := range linesToTranslate {
			prompt.WriteString(strings.Join(line.Text, " ") + "\n")
		}
	case 3:
		// Strategy 3: Explicitly request a specific format
		prompt.WriteString(fmt.Sprintf("You are a professional subtitle translator. Translate the following %s subtitles to %s. Your response should contain ONLY the translated subtitles, one per line:\n\n", originalLang, lang))
		for _, line := range linesToTranslate {
			prompt.WriteString("- " + strings.Join(line.Text, " ") + "\n")
		}
	default:
		// Default fallback strategy
		prompt.WriteString(fmt.Sprintf("Please translate these %s subtitles to %s as accurately as possible:\n\n", originalLang, lang))
		for _, line := range linesToTranslate {
			prompt.WriteString(strings.Join(line.Text, " ") + "\n")
		}
	}

	return prompt.String()
}

func processTranslatedResponse(response string, originalLines []modelSubtitles.ModelItemSubtitle) []modelSubtitles.ModelItemSubtitle {
	translatedLines := make([]modelSubtitles.ModelItemSubtitle, 0, len(originalLines))
	responseLines := strings.Split(strings.TrimSpace(response), "\n")

	// Match translated lines with original lines
	for i, translatedText := range responseLines {
		if i >= len(originalLines) {
			break
		}

		// Skip empty translations
		if strings.TrimSpace(translatedText) == "" {
			continue
		}

		// Create a new line with the translated text
		newLine := originalLines[i]
		newLine.Text = []string{strings.TrimSpace(translatedText)}
		translatedLines = append(translatedLines, newLine)
	}

	// If we didn't get enough translated lines, fill with original lines
	for i := len(translatedLines); i < len(originalLines); i++ {
		translatedLines = append(translatedLines, originalLines[i])
	}

	return translatedLines
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
