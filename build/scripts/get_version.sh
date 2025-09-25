#!/usr/bin/env bash

set -e

: "${GIT_CMD:=git}"

function _git {
    ${GIT_CMD} "$@"
}

function main {
    arg=$1
    if [[ "${arg}" == "app" || "${arg}" == "chart" ]]; then
        get_version_for $arg
    else
        echo "unknown '${arg}' type requested to show version (possible values: app, chart)"
        exit 1
    fi
}

function get_version_for {
    kind=$1
    version=""
    cur_commit="$(_git rev-parse --short=8 HEAD | cut -b 1-8)"
    cur_branch="$(_git branch --show-current)"
    # if we are on release branch - release versions should be prepared
    if [[ "${cur_branch}" =~ ^release- ]]; then
        last_tag="$(_git describe --abbrev=0 --tags)"
        commits_since_tag="$(_git rev-list --count "${last_tag}"..HEAD)"
        if [[ "${commits_since_tag}" != "0" ]]; then
            version=${echo $last_tag | sed -E "s/[0-9]+$/${commits_since_tag}/"}
        else
            version=${last_tag}
        fi
    else
        if [[ "${kind}" == "app" ]]; then
            if [[ "${cur_branch}" != "" ]]; then
                version="${cur_branch}-${cur_commit}"
            else
                version="dev-${cur_commit}"
            fi
        else
            chart_file="charts/pelagia-ceph/Chart.yaml"
            if [[ -f ${chart_file} ]]; then
                current_semantic_version=$(grep -E "^version:" ${chart_file} | cut -d: -f2 | tr -d ' ')
                if [[ "${cur_branch}" != "" ]]; then
                    version="${current_semantic_version}-${cur_branch//\//-}+${cur_commit}"
                else
                    version="${current_semantic_version}-dev+${cur_commit}"
                fi
            else
                echo "file '$(pwd)/${chart_file}' is not found"
                exit 1
            fi
        fi
    fi
    echo ${version}
}

main "$@"
