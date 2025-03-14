package main

import (
	"context"
	"fmt"

	"github.com/jonathanhecl/gollama"
	"github.com/jonathanhecl/subtitle-processor/subtitles"
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
	sub.LoadFilename("red-en.srt")

	fmt.Println(len(sub.Lines), sub.Format)

	// r, err := g.Chat(ctx, `Translate the following subtitles from English to Spanish: \n'\n`+
	// 	string(data)+
	// 	`\n'\n\nFormat: `+FORMAT_SYNTAX[format]+`\nIMPORTANT: Only change {{.Text}}.`)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	// fmt.Println(r.Content)

}
