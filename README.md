# 🔊Sub-Translation

A tool to translate subtitle files using Ollama models in local.

## Features

- Translate subtitle files
- Support for SRT and SSA files
- 100% local models (no internet connection required)
- Can translate every language supported by the model

## Requirements

- Ollama (https://ollama.com/)
- Ollama model (default: phi4)

## Usage

```sh
sub-translation -s=<source.srt> [-t=<target.srt>] [-o=<language>] [-l=<language>] [-m=<model>]
```

### Parameters

- `-s=<source.srt>`: Source subtitle file
- `-t=<target.srt>`: Target subtitle file (optional, default: source_translated.srt)
- `-o=<language>`: Original language (optional, default: English)
- `-l=<language>`: Target language (optional, default: Español neutro)
- `-m=<model>`: Ollama model (optional, default: phi4)

### Screenshots

![Screenshot](https://i.imgur.com/ba4fIJR.png)

![Screenshot](https://i.imgur.com/5k3wdWh.png)

## License

[MIT](LICENSE)

## Release

[Release](https://github.com/jonathanhecl/sub-translation/releases)
