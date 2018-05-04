#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

set -x

function main() {
    local fast

    fast="${1:-}"

    cd "$(dirname "$0")"

    if [ -z "${fast}" ] ; then
        rm -rf quickstart/content/posts
        go run json2md.go data quickstart/content/posts -l 500 -v
        echo 'Posts' > quickstart/content/posts/_index.md
    fi

    cd quickstart
    rm -rf public/*
    hugo --stepAnalysis
    cd -

    rm -rf /var/www/jaytaylor.com/public_html/hn
    find quickstart/public -type d -exec chmod 755 {} +
    mv quickstart/public /var/www/jaytaylor.com/public_html/hn
}

if [ "${BASH_SOURCE[0]}" = "${0}" ] ; then
    main "$@"
fi

