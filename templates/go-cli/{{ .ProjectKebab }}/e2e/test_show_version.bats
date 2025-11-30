setup() {
    load 'support/bats-support/load'
    load 'support/bats-assert/load'
}

@test "show version" {
    run ${EXECUTABLE} --version
    assert_success
    assert_output --partial '{{.ProjectKebab}} version'
}
