#!/bin/sh
cd "$(dirname "$0")"
cd ..

echo generating build info

VERSION=$(git describe --tags | cut -d "v" -f 2 | cut -d "-" -f 1)
BRANCH=$(git branch --show-current)
COMMIT=$(git log --format="%H" -n 1)

if [ -z "${VERSION}" ]; then
    VERSION="unknown"
fi

if [ -z "${BRANCH}" ]; then
    BRANCH="unknown"
fi

if [ -z "${COMMIT}" ]; then
    COMMIT="unknown"
fi

if ! [ -z "$(git status --porcelain)" ]; then
    COMMIT="uncommitted"
fi

echo writing contents of cmd/build-info.yaml
echo ---

cat <<EOF | tee cmd/build-info.yaml
version: ${VERSION}
branch: ${BRANCH}
commit: ${COMMIT}
date: $(date -u +%FT%T%Z)
EOF
echo ---

echo done writing contents of cmd/build-info.yaml
