#!/usr/bin/env bash

set -e

: "${GIT_CMD:=git}"

function _git {
    ${GIT_CMD} "$@"
}

function get_version {
    main_version=$1
    build_mode=$2
    version=""
    cur_commit="$(_git rev-parse --short=8 HEAD)"
    if [[ "${build_mode}" == "dev" ]]; then
        version="${main_version}-dev-${cur_commit}"
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
            version="${main_version}-custom-${cur_commit}"
        fi
    fi
    echo ${version}
}

get_version "$@"
