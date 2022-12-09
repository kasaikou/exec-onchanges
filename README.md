# exec-onchanges
exec-onchanges: CLI tool executing command on changing files detected, Built in Go.

```sh
$ exec-onchanges --help
exec-onchanges: Execute command on file changed and created

Usage: exec-onchanges (Options...) -- (Command and arguments...)
Example: exec-onchanges -i=**.go -e=.git -- gofmt -w '{{FILEPATH}}'

Options:
  -h, --help:              Display help (this is this)
  -f, --file=path/to/file: Filepath to configuration file (YAML)
  -i, --include=path/rule: Monitoring path rule (support '*', '**' wildcards)
  -e, --exclude=path/rule: Excluded Monitoring path rule (support '*', '**' wildcards)
```