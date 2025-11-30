setup() {
    load 'support/bats-support/load'
    load 'support/bats-assert/load'
}

@test "get name from user input and output concatenated text" {
    run ./e2e/test_root.exp ${EXECUTABLE}
    assert_success
    refute_output --partial 'FAIL'
}
