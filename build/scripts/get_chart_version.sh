#!/usr/bin/env bash

set -e

function main {
	chart_file="${1}"
	if [[ -z "${chart_file}" ]]; then
		echo "Chart file is not specified"
		exit 1
	fi
	semantic_version=$(grep version ${chart_file} | cut -d: -f2 | tr -d ' ')
	app_version=$(grep appVersion ${chart_file} | cut -d: -f2 | tr -d ' ' | sed "s/\//-/")
        if [[ -n "${GERRIT_BRANCH}" ]]; then
		if [[ "${GERRIT_BRANCH}" == "pure" ]]; then
			echo "${semantic_version}";
			exit 0
		fi
	fi
	echo "${semantic_version}-${app_version}"
}

main "$@"
