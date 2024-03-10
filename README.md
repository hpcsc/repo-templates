# repo-templates

Templates for some of my commonly used project setup

## Usage

- Install [boilerplate](https://github.com/gruntwork-io/boilerplate)
- Generate project using template: `boilerplate --template-url ./template-name --output-folder ./destination-folder --missing-config-action ignore`

## Templates

- `go-cli`: Template for Go CLI that uses [Taskfile](https://taskfile.dev/), [GoReleaser](https://goreleaser.com/), Github Actions
- `go-worker`: Template for Go Worker that uses [Taskfile](https://taskfile.dev/), Github Actions
- `asdf-plugin`: Template for ASDF plugins (with assumption that the target of the plugin has Github release available)
