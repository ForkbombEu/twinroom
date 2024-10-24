# Gemini

Gemini is a command-line interface (CLI) tool that allows you to execute slangroom contracts dynamically from specified folders. It provides an easy way to list and run `slangroom contracts`  as commands or expose them via HTTP in daemon mode.

## Installation

To use Gemini, you need Go installed on your system. If you don't have Go installed, you can download it from [golang.org](https://golang.org/dl/).

Clone the repository:

```bash
git clone https://github.com/ForkbombEu/gemini
```
Build the executable:
```bash
make build
```
## Usage

### List Command

To list all slangroom files in a specified directory, use the following command:

```bash
./out/bin/gemini list <folder>
```
### Run Command

To execute a specific slangroom file, use the following command:

```bash
out/bin/
./gemini run <folder> <file>
```
### Daemon Mode

Gemini can also run in daemon mode, exposing the slangroom files via an HTTP server. Use the -d or --daemon flag:

```bash
./out/bin/gemini -d <folder>
```
If only a folder is provided with the -d flag, Gemini will list the available slangroom files via HTTP.

### Examples

List all slang files in the examples folder:

```bash
./out/bin/gemini list examples
```

Run a specific slang file:

```bash
./out/bin/gemini run examples hello
```
Start the HTTP server to expose the slang files:

```bash
./out/bin/gemini -d examples
```


## 📝 Site docs

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite
```
