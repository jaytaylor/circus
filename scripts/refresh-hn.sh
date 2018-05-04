#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

user="${user:-jaytaylor}"

function favorites() {
    if [ -e favorites.json ] ; then
        echo 'INFO: updating existing favorites.json' 1>&2
        hn-favorites -v --user="${user}" --existing=favorites.json > favorites-latest.json
        mv favorites.json{,.bak}
        mv favorites{-latest,}.json
    else
        echo 'INFO: fetching new favorites.json' 1>&2
        hn-favorites -v --user="${user}" > favorites.json
    fi
}

function upvotes() {
    if [ -e upvotes.json ] ; then
        echo 'INFO: updating existing upvotes.json' 1>&2
        hn-upvotes -v --user="${user}" --password="$(base64 -d < ~/.jthn)" --existing=upvotes.json > upvotes-latest.json
        mv upvotes.json{,.bak}
        mv upvotes{-latest,}.json
    else
        echo 'INFO: fetching new upvotes.json' 1>&2
        hn-upvotes -v --user="${user}" --password="$(base64 -d < ~/.jthn)" > upvotes.json
    fi
}

if [ "${BASH_SOURCE[0]}" = "${0}" ] ; then
    # Only auto-run when being executed (and don't auto-run functions when being sourced).
    favorites
    upvotes
fi

