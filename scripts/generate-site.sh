#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset


function showHelp() {
    echo ".-------------------------------------------------------------------------." 1>&2
    echo "| Static site-generator tool                                              |" 1>&2
    echo ".---------------.------------------------------------------.--------------." 1>&2
    echo "| flag          | description                              | env-var      |" 1>&2
    echo "+---------------+------------------------------------------+--------------+" 1>&2
    echo "| -b <dir-path>   Huge base directory                        \$HUGO_DIR    |" 1>&2
    echo "|                 (default: \"quickstart\")                                 |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -f              Fast mode - skips content tagging and MD   \$FAST=true|1 |" 1>&2
    echo "|                 rendering (uses / requires stale data)                  |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -h, ?           This help document                                      |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -l <limit>      Maximum number of articles to process      \$LIMIT       |" 1>&2
    echo "|                 (default: 500)                                          |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -o <dir-path>   Output directory                           \$OUTPUT_DIR  |" 1>&2
    echo "|                 (default:                                               |" 1>&2
    echo "|                   /var/www/jaytaylor.com/public_html/hn)                |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -s <dir-path>   JSON data source directory                 \$SRC_DIR     |" 1>&2
    echo "|                                                                         |" 1>&2
    echo "| -v              Enable verbose output                      \$VERBOSE     |" 1>&2
    echo "|                                                                         |" 1>&2
    echo '`'"-------------'--------------------------------------------'--------------'" 1>&2
}


function main() {
    local OPT
    local OPTARG
    local OPTIND
    local fast
    local h
    local limit
    local hugoDir
    local outputDir
    local srcDir
    local verbose

    OPTIND=1

    fast="${FAST:-}"
    limit="${LIMIT:-500}"
    hugoDir="${HUGO_DIR:-quickstart}"
    srcDir="${SRC_DIR:-}"
    outputDir="${OUTPUT_DIR:-/var/www/jaytaylor.com/hn}"
    verbose="${VERBOSE:-}"
    v=''

    while getopts "fh?l:o:s:v" OPT; do
        case "${OPT}" in
        b)
            hugoDir="${OPTARG}"
            ;;
        f)
            fast='true'
            ;;
        h|\?)
            showHelp
            exit 0
            ;;
        l)
            limit="${OPTARG}"
            ;;
        o)
            outputDir="${OPTARG}"
            ;;
        s)
            srcDir="${OPTARG}"
            ;;
        v)
            verbose='true'
            v='-v'
            ;;
        *)
            echo "ERROR: unrecognized parameter \"${OPT}\"" 1>&2
        esac
    done

    shift $((OPTIND-1))

    [ "${1:-}" = '--' ] && shift

    echo "DEBUG: $0 Configuration" 1>&2
    echo "DEBUG: $(echo "${0}" | python -c 'import sys; sys.stdout.write("-" * len(sys.stdin.read()))')-------------" 1>&2
    echo "DEBUG:      fast: ${fast}" 1>&2
    echo "DEBUG:   hugoDir: ${hugoDir}" 1>&2
    echo "DEBUG:     limit: ${limit}" 1>&2
    echo "DEBUG: outputDir: ${outputDir}" 1>&2
    echo "DEBUG:    srcDir: ${srcDir}" 1>&2
    echo "DEBUG: $(echo "${0}" | python -c 'import sys; sys.stdout.write("-" * len(sys.stdin.read()))')-------------" 1>&2

    if [ -z "${hugoDir}" ] ; then
        echo 'ERROR: missing required parameter: -b <hugo-dir-path>'
        exit 1
    fi

    if [ -z "${outputDir}" ] ; then
        echo 'ERROR: missing required parameter: -o <output-dir>'
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
        go run "$(dirname "$0")/json2md.go" "${srcDir}" "${hugoDir}/content/posts" -l "${limit}" ${v}
        echo 'Posts' > "${hugoDir}/content/posts/_index.md"
    fi

    rm -rf "${hugoDir}/public/"*
    cd "${hugoDir}"
    hugo
    #--stepAnalysis
    cd -

    if [ -e "${outputDir}" ] ; then
        rm -rf "${outputDir}.old"
        mv "${outputDir}"{,.old}
    fi

    find "${hugoDir}/public" -type d -exec chmod 755 {} +
    mv "${hugoDir}/public" "${outputDir}"

    rm -rf "${outputDir}.old"
}

if [ "${BASH_SOURCE[0]}" = "${0}" ] ; then
    main "$@"
fi

