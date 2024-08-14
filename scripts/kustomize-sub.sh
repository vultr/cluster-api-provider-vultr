set -o errexit
set -o nounset
set -o pipefail

root=$(dirname "${BASH_SOURCE[0]}")
bin_dir="$root/../bin"

"$bin_dir/kustomize-v5.3.0" build "$1" | "$bin_dir/envsubst-v1.2.0"