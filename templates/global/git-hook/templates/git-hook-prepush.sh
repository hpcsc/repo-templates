#!/bin/bash

# this script defines useful functions to be used by prepush hook
# only below 2 functions are expected to be "public" and can be called by prepush hook, other functions are "private" to this script:
# - `fail_if_pushing_to_main`: prevent pushing directly to main branch
# - `validate`:
#   - check changes between local and remote commits and only run validation tasks if needed, .e.g. run shellcheck if changes contain shellscripts
#   - there are assumptions that the calling project already has 2 tasks: `test:shellcheck` and `hook:validate-go-changes`

NOT_EXISTING_SHA=0000000000000000000000000000000000000000

echo_red() {
    printf "\033[0;31m%s\033[0m\n" "$*"
}

echo_green() {
    printf "\033[0;32m%s\033[0m\n" "$*"
}

ensure_head_ref_exists() {
    local remote="${1}"
    if [ ! -f ".git/refs/remotes/${remote}/HEAD" ]; then
        # ${remote}/HEAD is created when the repo is cloned
        # if it doesn't exist for some reason, manually create and point it to either `main` or `master` so that we can deterministically check main branch
        if [ -f ".git/refs/remotes/${remote}/main" ]; then
            git symbolic-ref "refs/remotes/${remote}/HEAD" "refs/remotes/${remote}/main"
            echo_green "pointed ${remote}/HEAD to ${remote}/main"
        elif [ -f ".git/refs/remotes/${remote}/master" ]; then
            git symbolic-ref "refs/remotes/${remote}/HEAD" "refs/remotes/${remote}/master"
            echo_green "pointed ${remote}/HEAD to ${remote}/master"
        else
            echo_red "unable to determine default branch"
            exit 1
        fi
    fi
}

get_main_branch() {
  git branch -rl "*/HEAD" | sed "s/^.*\///g"
}

validate_changes_between_commits() {
    local local_commit=$1
    local remote_commit=$2
    changes=$(git diff "${local_commit}" "${remote_commit}" --name-only)
    if echo "${changes}" | grep -qi 'hooks/\|\.sh$'; then
        task test:shellcheck
    fi

    if echo "${changes}" | grep -q '\.go$'; then
        # assume this task exists if this function is used
        task hook:validate-go-changes
    fi
}

fail_if_pushing_to_main() {
    local remote=${1}
    ensure_head_ref_exists "${remote}"

    main_branch=$(get_main_branch)
    current_branch=$(git rev-parse --abbrev-ref HEAD)

    if [ "${current_branch}" = "${main_branch}" ]; then
        echo_red "pushing to main branch '${main_branch}' is not allowed"
        exit 1
    fi
}

validate() {
    local remote="${1}"
    local local_ref="${2}"
    local local_sha="${3}"
    local remote_ref="${4}"
    local remote_sha="${5}"

    if [ "${local_sha}" = "${NOT_EXISTING_SHA}" ]; then
        echo_green "deleting remote branch ${remote_ref} at sha ${remote_sha}, nothing to do"
        exit 0
    fi

    if [ "${remote_sha}" = "${NOT_EXISTING_SHA}" ]; then
        # pushing new local branch, run all validations
        echo_green "pushing new local branch ${local_ref} at sha ${local_sha}"
        git fetch "${remote}" "$(get_main_branch)"
        validate_changes_between_commits "${local_sha}" "$(get_main_branch)"
        exit 0
    fi

    echo_green "${local_ref}@${local_sha} -> ${remote_ref}@${remote_sha}"
    git fetch "${remote}" "${remote_ref}" # remote ref might have commits that we don't have in local
    validate_changes_between_commits "${local_sha}" "${remote_sha}"
}
