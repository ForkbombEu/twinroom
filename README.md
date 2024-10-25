# Gemini

Gemini is a command-line interface (CLI) tool that enables you to execute `slangroom contracts` , either embedded within the tool or dynamically from specified folders. It provides an easy way to list and run `slangroom contracts`  as commands or expose them via HTTP in daemon mode.
 Embedded files must be placed in the contracts folder to be recognized and executed by the CLI.

## Installation

To use Gemini, you need Go installed on your system. If you don't have Go installed, you can download it from [golang.org](https://golang.org/dl/).

Clone the repository:

```bash
git clone https://github.com/ForkbombEu/gemini
```
### Build the executable:
You can build the executable using either the go build command or the provided Makefile.
Using go build:

To build the binary with a custom name using go build, run:

```bash
go build -o <custom_name> .
```

Using make build:

```bash
make build BINARY_NAME=<custom_name>
```
If you want to specify a custom binary name, you can do so by passing the BINARY_NAME variable.

Replace <custom_name> with your desired binary name.
## Usage

### List Command

To list all slangroom files in a specified directory, use the following command:

```bash
./out/bin/gemini list <folder>
```
If you want to list only embedded filesin the contracts folder, simply run:

```bash
./out/bin/gemini list
```
### Run a file

To execute a specific slangroom file, use the following command:

```bash
out/bin/./gemini <folder> <file>
```

If the file is embedded, you can also run it directly by providing just the filename:


```bash
out/bin/./gemini  <file>
```

### Daemon Mode

Gemini can also run in daemon mode, exposing the slangroom files via an HTTP server. Use the -d or --daemon flag:

```bash
./out/bin/gemini -d <folder> <file>
```
If a folder is provided with the -d flag, Gemini will list the available slangroom files via HTTP.

```bash
./out/bin/gemini list  -d <folder>
```

### Examples

List all slang files in the examples folder:

```bash
./out/bin/gemini list examples
```

Run a specific slang file:

```bash
./out/bin/gemini examples hello
```
Start the HTTP server to expose the slang files:

```bash
./out/bin/gemini -d examples hello
```


## 📝 Site docs

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite
```
