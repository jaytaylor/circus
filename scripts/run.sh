#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset 

here="$(dirname "$0")"

"${here}/generate-site.sh" -s "${here}/../upvotes-data" -o "${here}/../quickstart" -l 100 $*
