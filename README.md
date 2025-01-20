# Twinroom

Twinroom is a command-line interface (CLI) tool that enables you to execute [slangroom contracts](https://dyne.org/slangroom) written in [Zencode language](https://dev.zenroom.org).

Contracts can be both embedded within the tool (compiled at build time) and added dynamically from specified folders (specified at runtime).

It also provides an easy way to list and run currently configured contracts both as CLI commands or as HTTP API endpoints served in daemon mode.

When dynamic run-time contract files have the same path as built-in embedded contracts, then the contracts placed in the contracts folder at build time have priority and will override the rest.

## Installation

To use Twinroom, you need Go installed on your system. If you don't have Go installed, you can download it from [golang.org](https://golang.org/dl/).

Clone the repository:

```bash
git clone https://github.com/ForkbombEu/Twinroom
```
### Embedding Files

To embed files in your project, place them inside the `contracts` folder. Only contracts with the `.slang` extension will be considered for embedding.

For example:
```
contracts/
├── example1.slang  
├── example2.slang  
└── subfolder/  
    └── nested.slang  
```
- Files such as `example1.slang` and `nested.slang` will be embedded.  
- Other file types or folders not related to .slang files will be ignored during the embedding process, except for JSON files associated with the contracts.

#### Customizing Embedded Folder
If you want to embed files from a different folder, you can now specify a custom directory when running the `make build` command.

To specify a custom folder for embedding files:
```bash
make build dir=path/to/contracts_folder
```
Where `path/to/contracts_folder` is the folder containing the `.slang` files you wish to use in place of the default `contracts` folder.

### How It Works

- **Before building**: The `make build` command will replace the contents of the `contracts` folder with the contents of the folder specified in the `dir` argument.
  
- **Building**: After the contents are replaced, the project will be built as usual.

- **After building**: The `contracts` folder is restored to its original state from a backup folder (`contracts_backup`).


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
./out/bin/twinroom list <folder>
```
If you want to list only embedded files in the contracts folder, simply run:

```bash
./out/bin/twinroom list
```
### Run a file

To execute a specific slangroom file, use the following command:

```bash
./out/bin/twinroom <folder> <file>
```

If the file is embedded in the `contracts` folder , you can also run it directly by providing just the filename:


```bash
./out/bin/twinroom  <file>
```

Or if it is in a subdir of `contracts`:

```bash
./out/bin/twinroom  <subdir> <file>
```

### Daemon Mode

Twinroom can also run in daemon mode, exposing the slangroom files via an HTTP server. Use the `--daemon` flag:

```bash
./out/bin/twinroom --daemon <folder> <file>
```
If a folder is provided with the `--daemon` flag and the list command, twinroom will list the available slangroom files via HTTP.

```bash
./out/bin/twinroom list  --daemon <folder>
```
### Adding Additional Data to Slangroom Contrats

Twinroom supports loading additional JSON-based data for each slangroom file. This data can be provided through optional JSON files with specific names, stored alongside the main slangroom file in the same directory. The parameters can be:

* `data`
* `keys`
* `extra`
* `context`
* `conf`

To add data for a specific slangroom file, create JSON files following the naming convention below:

```text
<filename>.<param>.json
```

Where:
 * `<filename>` is the name of your contract file.
 * `<param>` is one of the parameters listed above.

For example, if you have a file called `hello.slang`, you can provide additional data by creating files like:

```text
hello.data.json
hello.keys.json
hello.extra.json
```

### Command Arguments and Flags from `metadata.json`

In addition to the above parameters, Twinroom allows you to define custom arguments, flags, and environment variables for each embedded slangroom file using a metadata.json file. This file provides information on how to pass data to the contract through the CLI, including:

 * **Arguments**: Custom positional arguments for the command.
 * **Options**: Custom flags that can be passed to the command.
 * **Environment**: Key-value pairs of environment variables that are set dynamically when the command is executed.

 #### Structure of `metadata.json`

The metadata file is automatically read by Twinroom to generate appropriate arguments, flags, and environment variable settings when executing embedded contract files. A typical metadata.json structure might look like this:

```json
{
    "description": "Example of a command with different arguments and options",
    "arguments": [
        {
            "name": "<username>",
            "description": "user to login"
        },
        {
            "name": "[password]",
            "description": "password for user if required"
        }
    ],
    "options": [
        {
            "name": "-n, --name <name>"
        },
        {
            "name": "-s, --secret",
            "hidden": true
        },
        {
            "name": "-t, --timeout <delay>",
            "description": "timeout in seconds",
            "default": "60"
        },
        {
            "name": "-p, --port <number>",
            "description": "port number",
            "env": [
                "PORT"
            ]
        },
        {
            "name": "-d, --drink <size>",
            "description": "drink size",
            "choices": [
                "small",
                "medium",
                "large"
            ]
        },
        {
            "name": "-f, --file <file>",
            "description": "file to read if you pass - the stdin is read instead",
            "file": true,
            "rawdata": true
        },
    ],
        "environment": {
        "VAR1": "value1",
        "VAR2": "value2"
    }
}
```
#### Field Descriptions:

* **description**: A text description of the command, explaining its purpose or behavior.
* **arguments**:
    * ***name***: The name of the argument. Use angle brackets (`<arg>`) for required arguments and square brackets (`[arg]`) for optional ones.
    * ***description(optional)***: A brief explanation of what the argument represents or its purpose.
* **options**:
    * ***name***: The flag name(s), including shorthand (`-n`) and long-form (`--name`) options.
    * ***hidden (optional)***: If true, the flag is hidden from the help menu.
    * ***description (optional)***: A brief explanation of the flag’s purpose.
    * ***default (optional)***: The default value for the flag if not explicitly provided.
    * ***env (optional)***: A list of environment variable names that can be used as fallback values for the flag.
    * ***choices (optional)***: An array of allowed values for the flag, ensuring users provide a valid input.
    * ***file (optional)***:  If set to `true`, the flag requires a JSON file path. The file's contents will be added to the slangroom input data.
    * ***rawdata (optional)***:  If set to true alongside `file: true`, the contents of the file will be added as raw data, with the flag name serving as the key.
* **environment**:
    * For example, "environment": `{ "VAR1": "value1", "VAR2": "value2" }` will set the environment variables `VAR1=value1` and`VAR2=value2` during command execution.

All values provided through arguments and flags are added to the slangroom input data as key-value pairs in the format `"flag_name": "value"`. If a parameter is present in both the CLI input and the corresponding `filename.data.json` file, the CLI input will take precedence, overwriting the value in the JSON file.


### Examples

List all contracts in the examples folder:

```bash
./out/bin/twinroom list examples
```

Run a specific contract:

```bash
./out/bin/twinroom examples hello
```

Run a contract with arguments and flag:

```bash
out/bin/twinroom test param username -n myname -d small -t 100
```

Start the HTTP server to expose contract:

```bash
./out/bin/twinroom --daemon examples hello
```


## 📝 Site docs

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite
```
