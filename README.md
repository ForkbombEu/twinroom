<div align="center">

# Twinroom <!-- omit in toc -->

### Command-line interface (CLI) tool that enables you to execute [slangroom contracts](https://dyne.org/slangroom) written in [Zencode language](https://dev.zenroom.org). <!-- omit in toc -->

</div>

<p align="center">
  <a href="https://www.forkbomb.solutions/">
    <img src="https://forkbomb.solutions/wp-content/uploads/2023/05/forkbomb_logo_espressione.svg" width="170">
  </a>
</p>

## ✨ Twinroom features

Twinroom let you execute slangroom contracts in two ways:
* By embedding them within the tool (compiled at build time)
* Dynamically chosen from certain folders (specified at runtime)

Moreover it also provides an easy way to list and run currently configured contracts both as CLI commands or as HTTP API endpoints served in [daemon mode](#-daeomon-mode).

When embedded and dynamic loaded file will have the same path then
the embedded ones will be kept and dynamic one will not be loaded.

***

<div id="toc">

### 🚩 Table of Contents

- [🎮 Quick start](#-quick-start)
- [💾 Build](#-build)
- [🐣 Embedded contracts as executable commands](#-embedded-contracts-as-executable-commands)
- [🔮 Metadata file](#-metadata-file)
- [🗃️ Additional data to Slangroom contrats](#️-additional-data-to-slangroom-contrats)
- [😈 Daemon mode](#-daemon-mode)
- [📝 Site docs](#-site-docs)
- [🐛 Troubleshooting \& debugging](#-troubleshooting--debugging)
- [😍 Acknowledgements](#-acknowledgements)
- [👤 Contributing](#-contributing)
- [💼 License](#-license)
</div>



## 🎮 Quick start

To start using `twinroom` run the following commands

```sh
# download the binary
wget https://github.com/forkbombeu/twinroom/releases/latest/download/twinroom -O ~/.local/bin/twinroom && chmod +x ~/.local/bin/twinroom

# list embedded contracts (that now are commands!)
twinroom test --help
# run an embedded contracts
twinroom test hello

# or list contracts from a folder
git clone https://github.com/ForkbombEu/twinroom
twinroom list twinroom/contracts/test
```

**[🔝 back to top](#toc)**

---

## 💾 Build

To be able to build twinroom you need the following dependencies to be available in your PATH:
* go@1.23.4
* [slangroom-exec@latest](https://github.com/dyne/slangroom-exec)

You can install them by hand or use [mise](https://mise.jdx.dev/) and run `mise install` in the root of the repository.

You can build the executable following the steps:

```bash
# clone the repository and enter in it
git clone https://github.com/ForkbombEu/twinroom
cd twinroom

# if you decided to use mise now run: mise install

# build the executable
make build
```

The binary will be found in `./out/bin/twinroom`.

### ✒️ Build with a custom name

The executable can also have a custom that can be specified at build time:
```sh
make build BINARY_NAME=<custom_bin_name>
```

### 📁 Build with custom embedded files

By default Twinroom will consider only the files or folder inside the
`contracts` folder, if you want to specify a different folder or use more
then one you can do it by creating the `extra_dir.json` file that contains
the path to the folders you want to embed, like:

```json
{
    "paths": [
        "path/to/first/folder/to/embed",
        "path/to/second/folder/to/embed",
        "as/many/path/as/you/want"
    ]
}
```

The files that twinroom will embed are the `.slang` files, *i.e.* the
contracts, and the JSON file associated with it (keys, data, conf, metadata, ...).
All the other files will be ignored.

**[🔝 back to top](#toc)**

---
## 🐣 Embedded contracts as executable commands

What happen to the contracts that you embed?

Suppose that we build the executable without changing name, *i.e.* it is `twinroom`, and using the default folder as target.
If we then run
```sh
./out/bin/twinroom --help
```
we will see that under available commands, there are
```sh
Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List all contracts in the folder or list embedded contracts if no folder is specified
  test        Commands for files in test
```

while *completion*, *help* and *list* are from twinroom source code **test** is the folder inside `contracts` that we have embedded, indeed if we run
```sh
./out/bin/twinroom test --help
```
we will have a new set of available commands
```sh
Available Commands:
  broken          Execute the embedded contract broken
  env             Example of a command that can use env variables
  execute_zencode Execute the embedded contract execute_zencode
  hello           Execute the embedded contract hello
  introspection   Execute the embedded contract introspection
  param           Example of a command with different arguments and options
  read_from_path  read a file content from a path
  stdin           read a file content from pipe stdin or if a filename given the content of the file
  test            Execute the embedded contract test
  withschema
```
If you look in the `./contracts/test` folder you will see that for each of this command there is a `*.slang` file and running one of this command
will result in running the slangroom contract behind it, *e.g.*
```sh
./out/bin/twinroom test hello
```
will result in
```json
{"output":["Hello_from_embedded!"]}
```

In this case no input was required to run the `hello` command, but when an input from the user side is required this can be specified in the [metdata file](#-metadata-file).

**[🔝 back to top](#toc)**

---
## 🔮 Metadata file

Twinroom allows you to define custom arguments, flags, and environment variables for each embedded slangroom file using a `<contract_name>.metadata.json` file.
This file provides information on how to pass data to the contract through the CLI, including:

 * **Arguments**: Custom positional arguments for the command.
 * **Options**: Custom flags that can be passed to the command.
 * **Environment**: Key-value pairs of environment variables that are set dynamically when the command is executed.

 ### 🤖 Structure of `metadata.json`

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

where:
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

**[🔝 back to top](#toc)**

---
## 🗃️ Additional data to Slangroom contrats

Along with metadata, Twinroom supports loading additional JSON-based data for each slangroom file.
This data can be provided through optional JSON files with specific names, stored alongside the main slangroom file in the same directory. The parameters can be:

* `data`
* `keys`
* `extra`
* `context`
* `conf`

To add data for a specific slangroom file, create JSON files following the naming convention below:

```text
<contract_name>.<param>.json
```

Where:
 * `<contract_name>` is the name of your contract file.
 * `<param>` is one of the parameters listed above.

For example, if you have a file called `hello.slang`, you can provide additional data by creating files like:

```text
hello.data.json
hello.keys.json
hello.extra.json
```

**[🔝 back to top](#toc)**

---
## 😈 Daemon mode

Twinroom can also run in daemon mode, exposing the slangroom files via an HTTP server. Use the `--daemon` flag:

```bash
./out/bin/twinroom --daemon <folder> <file>
```
If a folder is provided with the `--daemon` flag and the list command, twinroom will list the available slangroom files via HTTP.

```bash
./out/bin/twinroom list  --daemon <folder>
```

**[🔝 back to top](#toc)**

---
## 📝 Site docs

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest && pkgsite
```

## 🐛 Troubleshooting & debugging

Availabe bugs are reported via [GitHub issues](https://github.com/forkbombEu/twinroom/issues).

**[🔝 back to top](#toc)**

---
## 😍 Acknowledgements

Copyright © 2023-2025 by [Forkbomb BV](https://www.forkbomb.solutions/), Amsterdam

**[🔝 back to top](#toc)**

---
## 👤 Contributing

1.  🔀 [FORK IT](../../fork)
2.  Create your feature branch `git checkout -b feature/branch`
3.  Commit your changes `git commit -am 'feat: New feature\ncloses #398'`
4.  Push to the branch `git push origin feature/branch`
5.  Create a new Pull Request `gh pr create -f`
6.  🙏 Thank you


**[🔝 back to top](#toc)**

---
## 💼 License

    Twinroom - Create CLI tools from slangroom contracts
    Copyleft 🄯 2024-2025 The Forkbomb Company, Amsterdam

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as
    published by the Free Software Foundation, either version 3 of the
    License, or (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.

**[🔝 back to top](#toc)**

<!--
### How It Works

- **Before building**: The `make build` command reads the `extra_dir.json` file to retrieve the paths to the directories containing the `.slang` files. The contents of the `contracts` folder will be replaced with the contents of the specified directories.

- **Building**: After the contents are replaced, the project will be built as usual. If no `extra_dir.json` file is found or if it does not specify any paths, the default `contracts` folder is used.

- **After building**: Once the build is complete, the `contracts` folder is restored to its original state.
-->
