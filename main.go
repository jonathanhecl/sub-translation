package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jonathanhecl/gollama"
)

const (
	FORMAT_SRT = "srt"
	FORMAT_SSA = "ssa"
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

	data, format, err := loadFile("red-en.srt")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(data), format)
}

func loadFile(filename string) ([]byte, string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}

	if strings.HasSuffix(filename, ".srt") {
		return data, FORMAT_SRT, nil
	} else if strings.HasSuffix(filename, ".ssa") {
		return data, FORMAT_SSA, nil
	}

	return data, "", nil
}
