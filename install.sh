#!/usr/bin/env bash

set -Eeuo pipefail

BINDIR="${BINDIR:-/usr/local/bin}"

REPO_OWNER="tukaelu"
PROG_NAME="zgsync"

if ! \curl --version >/dev/null 2>&1; then
  echo "curl command is required to install ${PROG_NAME}" 2>&1
  echo "Please install curl and try again." 2>&1
  exit 1
fi

get_goos() {
  os=$(uname -s)
  case $os in
  "Linux")
    echo "linux"
    ;;
  "Darwin")
    echo "darwin"
    ;;
  "FreeBSD")
    echo "freebsd"
    ;;
  *)
    echo "unsupported OS: ${os}"
    exit 1
    ;;
  esac
}

get_goarch() {
  arch=$(uname -m)
  case $arch in
  "amd64" | "x86_64")
    echo "amd64"
    ;;
  "arm64" | "aarch64")
    echo "arm64"
    ;;
  *)
    echo "unsupported arch: ${arch}"
    exit 1
    ;;
  esac
}

get_latest_version() {
  version=$(
    curl -fsSL https://api.github.com/repos/${REPO_OWNER}/${PROG_NAME}/releases/latest |
      grep tag_name |
      cut -d: -f2-3 |
      sed -e 's/["v, ]//g'
  )
  echo "${version}"
}

get_ext() {
  case $goos in
  "linux" | "freebsd")
    echo ".tar.gz"
    ;;
  "darwin")
    echo ".zip"
    ;;
  *)
    echo "unsupported OS: ${os}"
    exit 1
    ;;
  esac
}

do_extract() {
  local path=$1
  local fname=$2
  local ext=$3
  case ${ext} in
  ".tar.gz")
    tar -C "${path}" -xzf "${path}/${fname}"
    ;;
  ".zip")
    unzip -d "${path}" "${path}/${fname}"
    ;;
  *)
    echo "unsupported ext: ${ext}"
    exit 1
    ;;
  esac
}

goos=$(get_goos)
goarch=$(get_goarch)
ext=$(get_ext)
latest_version=$(get_latest_version)

fname="${PROG_NAME}_${latest_version}_${goos}_${goarch}${ext}"
release_url="https://github.com/${REPO_OWNER}/${PROG_NAME}/releases/download/v${latest_version}/${fname}"

tmpdir=$(mktemp -d)

echo "Downloading ${fname} from ${release_url} ... "
curl -fSL -o "${tmpdir}/${fname}" "${release_url}"
echo "done."

echo -n "Extracting ... "
do_extract "${tmpdir}" "${fname}" "${ext}"
echo "done."

echo -n "Installing ... "
mv "${tmpdir}/${PROG_NAME}" "${BINDIR}/${PROG_NAME}"
chmod +x "${BINDIR}/${PROG_NAME}"
echo "done."
