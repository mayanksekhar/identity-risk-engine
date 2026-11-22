#!/usr/bin/env bash
# Fetch the pinned Mons dataset files for Cohort A.
# Dataset: "A dataset of GitHub Actions workflow histories", version 2026-05-22,
# Zenodo record 20340547, licence CC BY 4.0. Cite Cardoen, Mens, Decan (MSR 2024).
# Zenodo double compresses the .gz files and the listed md5 refers to the original
# file, so this script accepts either the file as served or its once decompressed form.
# Data goes outside the repository by default. Never commit it.
set -euo pipefail

RECORD=20340547
DATA_DIR="${DATA_DIR:-$HOME/data/gha-workflows}"
MIN_FREE_KB=$((6 * 1024 * 1024))

declare -A MD5=(
  [repositories.csv.gz]=a345384e924083ae4da327e4fdf3af74
  [workflows.csv.gz]=b557752520849d50a99d3750a8f62b68
  [workflows.tar.gz]=6b1f38ba8327734f8d155f25e3011507
)
ORDER=(repositories.csv.gz workflows.csv.gz workflows.tar.gz)

mkdir -p "$DATA_DIR"
free_kb=$(df -Pk "$DATA_DIR" | awk 'NR==2 {print $4}')
if (( free_kb < MIN_FREE_KB )); then
  echo "Need about 6 GB free in $DATA_DIR, found $((free_kb / 1024 / 1024)) GB" >&2
  exit 1
fi

md5_of() { md5sum "$1" | cut -d' ' -f1; }

# Installs the file and returns 0 if the part file verifies, else returns 1.
accept_part() {
  local want=$1 out=$2
  if [[ "$(md5_of "$out.part")" == "$want" ]]; then
    mv "$out.part" "$out"
    return 0
  fi
  if gzip -dc "$out.part" 2>/dev/null | tee "$out.inner" | md5sum | grep -q "$want"; then
    mv "$out.inner" "$out"
    rm -f "$out.part"
    echo "  (file was double compressed, kept the original form)"
    return 0
  fi
  rm -f "$out.inner"
  return 1
}

for name in "${ORDER[@]}"; do
  want=${MD5[$name]}
  out="$DATA_DIR/$name"

  if [[ -f "$out" && "$(md5_of "$out")" == "$want" ]]; then
    echo "ok, already verified: $name"
    continue
  fi

  echo "downloading $name"
  url="https://zenodo.org/records/${RECORD}/files/${name}?download=1"
  if ! curl -L --fail --retry 5 -C - -o "$out.part" "$url"; then
    # A rerun on a complete part file makes the server answer 416, so verify what we have.
    [[ -f "$out.part" ]] || { echo "download failed: $name" >&2; exit 1; }
  fi

  if accept_part "$want" "$out"; then
    echo "ok: $name"
  else
    echo "MD5 MISMATCH for $name. If the download was cut short, rerun to resume." >&2
    echo "To start over, delete $out.part" >&2
    exit 1
  fi
done

echo "all files verified in $DATA_DIR"
