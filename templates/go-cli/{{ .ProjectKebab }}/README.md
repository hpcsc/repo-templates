# {{.ProjectKebab}}

## Goreleaser

- Run goreleaser in local: `task release:local`. This will generate a snapshot build under `./dist`
- Create a release:

```shell
git tag vX.X.X
git push origin vX.X.X
```

This will trigger release workflow which will create a Github Release with binaries for MacOS and Linux

Release workflow can also be triggered from Github Actions manually using workflow dispatch. In this mode, the workflow just creates a snapshot build, similar to `task release:local`, no Github Release will be created.

## E2E Test

This project uses [bats](https://bats-core.readthedocs.io) and [expect](https://core.tcl-lang.org/expect/index) for e2e tests against the built binary:

- Bats is used as the primary test framework. All non-interactive test cases can be written in Bats
- For interactive test cases (.i.e program requiring user input), Bats tests run expect scripts, which spawn the program under test and simulate user input.
Expect scripts are expected to output `OK` when successful and `FAIL` when not. Bats tests assert for absence of `FAIL` keyword from expect script output
