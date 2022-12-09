# exec-onchanges
exec-onchanges: CLI tool executing command on changing files detected, Built in Go.

```sh
> exec-onchanges -e=.git -e=.github -e=.devcontainer -- echo "{{FILEPATH}}"
```