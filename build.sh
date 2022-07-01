#!/bin/bash

# Copyright 2021 RELEX Oy
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Environments:
#   CGO_ENABLED: enable CGO or not; default 0
#   GO_LDFLAGS: to be passed in -ldflags
#
# Command-line arguments:
#   passed to go build

set -o pipefail

GOPATH=$(go env GOPATH) || exit 1
CGO_ENABLED=${CGO_ENABLED:-0}
test -d BUILD || { echo "missing BUILD dir"; exit 1; }

# Generate and check go inline reports by special "xx:inline" comment
# e.g. func (s LogSchema) GetFieldName(index int) string { // xx:inline

INLINE_REPORT_FILE="BUILD/go-inline-$$.txt"

collect_inline_hints() {
    # e.g. base/logschema.go:84:6: can inline LogSchema.GetFieldName
    cat | grep ': can inline ' > $INLINE_REPORT_FILE
    return 0
}

hide_inline_hints() {
    cat | grep -E -v '(can inline|escapes|ignoring self-assignment|inlining|leaking param|moved to heap|not escape)' | grep -E -v '^# '
    return 0
}

function finish {
    rm $INLINE_REPORT_FILE
}
trap finish EXIT

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [[ $OS == "darwin" ]]
then
    GO_LDFLAGS_FULL="$GO_LDFLAGS" # "-static" is not supported on OS/X when linker is used
else
    GO_LDFLAGS_FULL="$GO_LDFLAGS -extldflags -static"
fi

GO111MODULE=on CGO_ENABLED=$CGO_ENABLED go build -gcflags "./...=-m -l=4" -ldflags "$GO_LDFLAGS_FULL" -o BUILD "$@" 2>&1 | tee >(collect_inline_hints) | hide_inline_hints
GOBUILD_EXIT=$?
if [[ "$GOBUILD_EXIT" != "0" ]]
then
    exit $GOBUILD_EXIT
fi

INLINE_FAIL=0

IFS=$'\n'
for LINE in $(find . -name "*.go" -exec grep -Hn "// xx:inline" '{}' ';')
do
    # e.g. ./base/logschema.go:84:func (s *xLogSchema) GetFieldName(index int) string { // xx:inline
    LINE=${LINE#./}
    LINE=${LINE%(*}
    LHEADER=${LINE%:*}
    if $(grep -q "^$LHEADER" $INLINE_REPORT_FILE)
    then
        echo $LINE is inlined
    else
        echo -e $LINE is NOT inlined "\t<<<<<<<<<< <<<<<<<<<< <<<<<<<<<<" >&2
        INLINE_FAIL=1
    fi
done

if [[ $INLINE_FAIL != "0" ]]
then
    exit 1
fi
