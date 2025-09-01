#!/usr/bin/env bash

set -e

: "${GIT_CMD:=git}"

function _git {
    ${GIT_CMD} "$@"
}

function main {
    cur_commit="$(_git rev-parse --short=8 HEAD | cut -b 1-8)"
    if [[ -n "${GERRIT_BRANCH}" ]]; then
        cur_branch="${GERRIT_BRANCH}"
        if [[ "${GERRIT_EVENT_TYPE}" == "change-merged" ]]; then
            if [[ "${cur_branch}" == "master" ]]; then
                last_tag="master"
                commits_since_tag="$(_git rev-list --count HEAD)"
            else
                last_tag="$(_git describe --abbrev=0 --tags)"
                commits_since_tag="$(_git rev-list --count "${last_tag}"..HEAD)"
            fi
        else
            gerrit_branch=$(echo "${GERRIT_BRANCH}" | sed 's/\//-/')
            last_tag="dev-${gerrit_branch}"
            commits_since_tag="$(_git rev-list --count "origin/${GERRIT_BRANCH}"..HEAD)-$(_git rev-parse --short HEAD)"
        fi
    else
        last_tag="dev"
        commits_since_tag="$(_git rev-list --count HEAD)-$(_git rev-parse --short HEAD)"
    fi

    if [[ -n "${1}" ]]; then
        chart_tag="$(echo "${1}" | sed 's/\.[0-9]*$//')"
        echo "${chart_tag}.${commits_since_tag}"
    else
        # case for code version
        echo "${last_tag}-${cur_commit}"
    fi
}

main "$@"
