# Diffy

Diffy is a terminal-first Git diff viewer for local changes, branch comparisons, stashes, recent commits, file history, and single commits.

It is keyboard-first, but supports mouse interactions where the terminal allows them.

## Requirements

- Go 1.24 or newer
- Git
- A terminal with color support

## Install

Clone the repo and install from the project directory:

```bash
git clone https://github.com/KunalHS/diffy.git
cd diffy
go install .
```

`go install .` writes the `diffy` binary into your Go bin directory, usually:

```bash
$(go env GOPATH)/bin
```

Make sure that directory is on your `PATH`.

For zsh, add this to `~/.zshrc`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.zshrc
```

Verify the install:

```bash
which diffy
diffy --help
```

## Update

From the cloned repo:

```bash
git pull
go install .
```

## Common Commands

```bash
diffy
```

Open local uncommitted changes.

```bash
diffy main
diffy compare main
diffy review main
```

Compare the current branch against `main`.

```bash
diffy feature-branch live
diffy compare feature-branch live
```

Compare one branch into another.

```bash
diffy stash
```

Inspect and manage stashes.

```bash
diffy history
diffy history --file path/to/file
```

Inspect file history.

```bash
diffy commit abc123
```

Inspect one commit.

## Useful Options

```bash
diffy --summary
diffy --raw
diffy --unified
diffy --split
diffy --file path/to/file
diffy --path path/to/file
```

`--path` is an alias for `--file`.

## Config

Diffy reads config from:

```bash
~/.config/diffy/config.yaml
```

Supported settings:

```yaml
layout: split
sidebar: true
```

Valid layout values:

```yaml
layout: split
layout: unified
```

Valid sidebar values:

```yaml
sidebar: true
sidebar: false
```

Diffy also keeps session state next to the config file:

```bash
~/.config/diffy/state.json
```

Explicit config values win over remembered state.
