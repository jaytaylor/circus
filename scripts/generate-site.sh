#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset


function showHelp() {
    echo ".------------------------------------------------------------------------." 1>&2
    echo "| Static site-generator tool                                             |" 1>&2
    echo ".---------------.------------+----------------------------.--------------." 1>&2
    echo "| flag          | description                             | env-var      |" 1>&2
    echo "+---------------+-----------------------------------------+--------------+" 1>&2
    echo "| -f             Fast mode - skips content tagging and MD   \$fast=true|1 |" 1>&2
    echo "|                rendering (uses / requires stale data)                  |" 1>&2
    echo "|                                                                        |" 1>&2
    echo "| -h, ?          This help document                                      |" 1>&2
    echo "|                                                                        |" 1>&2
    echo "| -l <limit>     Maximum number of articles to process      \$limit       |" 1>&2
    echo "|                (default: 500)                                          |" 1>&2
    echo "|                                                                        |" 1>&2
    echo "| -o <dir-path>  Huge base directory                        \$hugoDir     |" 1>&2
    echo "|                                                                        |" 1>&2
    echo "| -s <dir-path>  JSON data source directory                 \$srcDir      |" 1>&2
    echo "|                                                                        |" 1>&2
    echo "| -v             Enable verbose output                      \$verbose     |" 1>&2
    echo "|                                                                        |" 1>&2
    echo '`'"-------------'-------------------------------------------'--------------'" 1>&2
}


function main() {
    local OPT
    local OPTARG
    local OPTIND
    local fast
    local h
    local limit
    local hugoDir
    local srcDir
    local verbose

    OPTIND=1

    fast="${fast:-}"
    limit="${limit:-500}"
    verbose="${verbose:-}"

    while getopts "fh?l:o:s:v" OPT; do
        case "${OPT}" in
        f)
            fast=true
            ;;
        h|\?)
            showHelp
            exit 0
            ;;
        l)
            limit="${OPTARG}"
            ;;
        o)
            hugoDir="${OPTARG}"
            ;;
        s)
            srcDir="${OPTARG}"
            ;;
        v)
            verbose=true
            ;;
        *)
            echo "ERROR: unrecognized parameter \"${OPT}\"" 1>&2
        esac
    done

    shift $((OPTIND-1))

    [ "${1:-}" = '--' ] && shift

    echo 'DEBUG: Configuration' 1>&2
    echo 'DEBUG: -------------' 1>&2
    echo "DEBUG:    fast=${fast}" 1>&2
    echo "DEBUG:   limit=${fast}" 1>&2
    echo "DEBUG: hugoDir=${hugoDir}" 1>&2
    echo "DEBUG:  srcDir=${srcDir}" 1>&2
    echo 'DEBUG: -------------' 1>&2

    if [ -z "${hugoDir}" ] ; then
        echo 'ERROR: missing required parameter: -o <hugo-dir-path>'
        exit 1
    fi

    if [ -z "${srcDir}" ] ; then
        echo 'ERROR: missing required parameter: -s <src-data-dir-path>'
        exit 1
    fi

    if [ "${verbose}" = 'true' ] || [ "${verbose}" = '1' ] ; then
        set -x
    fi

    if [ "${fast:-}" != 'true' ] && [ "${fast:-}" != '1' ] ; then
        rm -rf "${hugoDir}/content/posts"
        go run "$(dirname "$0")/json2md.go" "${srcDir}" "${hugoDir}/content/posts" -l "${limit}" -v
        echo 'Posts' > "${hugoDir}/content/posts/_index.md"
    fi

    rm -rf "${hugoDir}/public/"*
    cd "${hugoDir}"
    hugo --stepAnalysis
    cd -

    if [ -e /var/www/jaytaylor.com/public_html/hn ] ; then
        rm -rf /var/www/jaytaylor.com/public_html/hn.old
        mv /var/www/jaytaylor.com/public_html/hn{,.old}
    fi

    find "${hugoDir}/public" -type d -exec chmod 755 {} +
    mv "${hugoDir}/public" /var/www/jaytaylor.com/public_html/hn

    rm -rf /var/www/jaytaylor.com/public_html/hn.old
}

if [ "${BASH_SOURCE[0]}" = "${0}" ] ; then
    main "$@"
fi

