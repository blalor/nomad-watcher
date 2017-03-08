#!/usr/bin/env bash
set -e -u -o pipefail

## must be run as root, or "acbuild run" will fail

## sudo fucks with our PATH, via secure_path
export PATH=/usr/local/bin:$PATH

acbuild="acbuild --debug --no-history"

if [ $# -ne 2 ]; then
    echo "usage: $0 <aci name> <version>"
    exit 1
fi

aci_name="${1}"
aci_version="${2}"

cur_dir=${PWD}
dest_dir="${cur_dir}/work"
mkdir -p "${dest_dir}"

function cleanup() {
    EXIT=$?
    
    $acbuild end
    
    exit ${EXIT}
}

trap cleanup EXIT

$acbuild begin

$acbuild annotation add created "$( date --rfc-3339=ns | tr ' ' 'T' )"
$acbuild set-name "${aci_name}"

$acbuild label add version "${aci_version}"
$acbuild label add os linux
$acbuild label add arch amd64

$acbuild copy stage/nomad-watcher /nomad-watcher
$acbuild set-exec -- /nomad-watcher

mkdir -p "$( dirname "${dest_dir}/${aci_name}" )"
$acbuild write "${dest_dir}/${aci_name}"

## generate XML (aieeee!) describing what we just built
## this is mainly for GoCD, so we can set properties based on this artifact
{
    echo '<?xml version="1.0" encoding="utf-8"?>'
    echo '<aci>'
    echo "    <name>${aci_name}</name>"
    echo "    <version>${aci_version}</version>"
    echo '</aci>'
} > "${dest_dir}/aci.xml"

## also, since I don't yet have a handle on how these dependencies are going to
## be chained, capture the manifest.
$acbuild cat-manifest > "${dest_dir}/manifest"

## we're probably root, anyway, but check just in case
if [ "${EUID}" -eq 0 ]; then
    ## ensure the calling user can remove the generated files
    chown -R "$( stat --format='%U:%G' . )" "${dest_dir}"
fi
