#!/usr/bin/env bash

set -e

: "${GIT_CMD:=git}"

function _git {
    ${GIT_CMD} "$@"
}

function get_app_version {
    main_version=$1
    dev_version=$2
    version=""
    if [[ -n "${dev_version}" ]]; then
        version="${main_version}-${dev_version}"
    else
        last_tag="$(_git describe --abbrev=0 --tags --always)"
        # check we are good for release
        if [[ "${last_tag}" == "${main_version}" ]]; then
            commits_since_tag="$(_git rev-list --count "${last_tag}"..HEAD)"
            if [[ "${commits_since_tag}" != "0" ]]; then
                version="$(echo $last_tag | sed -E "s/[0-9]+$/${commits_since_tag}/")"
            else
                version="${last_tag}"
            fi
        else
            cur_commit="$(_git rev-parse --short=10 HEAD)"
            version="${main_version}-custom-${cur_commit}"
        fi
    fi
    echo ${version}
}

function get_git_version {
    version=$(_git rev-parse --short=10 HEAD)
    echo $version
}

if [ "$#" -lt 1 ]; then
    get_git_version
else
    get_app_version "$@"
fi
