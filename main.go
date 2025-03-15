package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jonathanhecl/gollama"
	"github.com/jonathanhecl/subtitle-processor/subtitles"
	modelSubtitles "github.com/jonathanhecl/subtitle-processor/subtitles/models"
)

var (
	version = "0.1.0"
)

func main() {
	fmt.Println("Sub-Translation v" + version)

	ctx := context.Background()
	g := gollama.New("phi4")
	g.Verbose = true
	if err := g.PullIfMissing(ctx); err != nil {
		fmt.Println("Error:", err)
		return
	}

	sub := subtitles.Subtitle{}
	err := sub.LoadFile("./red-en.srt")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if len(sub.Lines) < 3 {
		fmt.Println("Not enough lines in subtitle file")
		return
	}

	translate := subtitles.Subtitle{}
	translate.Filename = "./red-es.srt"
	translate.Format = sub.Format
	translate.Lines = make([]modelSubtitles.ModelItemSubtitle, 0)

	// LOGIC:
	// 1. Iterate over each subtitle line sending to Gollama
	//    a. First 3 lines on the first iteration to translate
	//    b. Then next line, with the previous 3 lines translated
	// 2. Save the translated lines to a new subtitle file

	// Example: Gollama Chat
	// g.Chat(ctx, `Translate the following subtitles from English to Spanish: \n'....

	for i := 0; i < len(sub.Lines); i++ {
		var prompt string
		var linesToTranslate [][]string

		if i == 0 {
			// First iteration: take first 3 lines
			if i+2 < len(sub.Lines) {
				for j := i; j < i+3; j++ {
					linesToTranslate = append(linesToTranslate, sub.Lines[j].Text)
				}
			} else {
				for j := i; j < len(sub.Lines); j++ {
					linesToTranslate = append(linesToTranslate, sub.Lines[j].Text)
				}
			}
		} else {
			// Take previous 3 translated lines + current line for context
			prevStart := len(translate.Lines) - 3
			if prevStart < 0 {
				prevStart = 0
			}
			prompt = "Previous context in Spanish:\n"
			for j := prevStart; j < len(translate.Lines); j++ {
				prompt += strings.Join(translate.Lines[j].Text, "\n") + "\n"
			}
			prompt += "\nNow translate this line from English to Spanish maintaining the same style:\n"
			linesToTranslate = append(linesToTranslate, sub.Lines[i].Text)
		}

		// Build the translation prompt
		prompt += "Translate the following subtitles from English to Spanish:\n"
		for _, textLines := range linesToTranslate {
			prompt += strings.Join(textLines, "\n") + "\n"
		}

		// Get translation from Gollama
		response, err := g.Chat(ctx, prompt)
		if err != nil {
			fmt.Printf("Error translating line %d: %v\n", i+1, err)
			continue
		}

		// Split response into lines and create new subtitles
		translatedLines := strings.Split(strings.TrimSpace(response.Content), "\n")
		for j, translatedText := range translatedLines {
			if j+i >= len(sub.Lines) {
				break
			}
			newLine := sub.Lines[j+i]
			newLine.Text = []string{strings.TrimSpace(translatedText)}
			translate.Lines = append(translate.Lines, newLine)
		}

		// Skip the next 2 lines on first iteration since we already translated them
		if i == 0 && len(linesToTranslate) > 1 {
			i += 2
		}
	}

	err = translate.SaveFile(translate.Filename)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
