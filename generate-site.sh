#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

set -x

function main() {
    cd "$(dirname "$0")"

    rm -rf quickstart/content/posts
    go run json2md.go data quickstart/content/posts
    echo 'Posts' > quickstart/content/posts/_index.md

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

