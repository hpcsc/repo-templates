# repo-templates

Cookiecutter templates for some of my commonly used project setup

## Usage

- Install cookiecutter: `python3 -m pip install --user cookiecutter`
- Generate project using template: `cookiecutter -f ./template-name -o ./destination-folder`

```shell
# Examples
cookiecutter -f ./go-cli # assuming current directory is a clone of this repository
cookiecutter gh:hpcsc/repo-templates --directory go-cli # or directly from github
```
## Templates

- `go-cli`: Template for Go CLI that uses [Taskfile](https://taskfile.dev/), [GoReleaser](https://goreleaser.com/), Github Actions
- `asdf-plugin`: Template for ASDF plugins (with assumption that the target of the plugin has Github release available)
