# {{cookiecutter.name}}

## Goreleaser

- Run goreleaser in local: `task release:local`. This will generate a snapshot build under `./dist`
- Create a release:

```shell
git tag vX.X.X
git push origin vX.X.X
```

This will trigger release workflow which will create a Github Release with binaries for MacOS and Linux

Release workflow can also be triggered from Github Actions manually using workflow dispatch. In this mode, the workflow just creates a snapshot build, similar to `task release:local`, no Github Release will be created.
