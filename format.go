package main

const (
	FORMAT_SRT = "srt"
	FORMAT_SSA = "ssa"
)

var (
	FORMAT_SYNTAX = map[string]string{
		FORMAT_SRT: `{{.Index}}
{{.StartTime}} --> {{.EndTime}}
{{.Text}}
`,
		FORMAT_SSA: `{{.Index}}
{{.StartTime}} --> {{.EndTime}}
{{.Text}}
`,
	}
)
