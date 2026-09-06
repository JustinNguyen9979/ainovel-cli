#!/bin/sh
set -eu

dist="${1:-dist}"
version="${2:?version is required}"

checksum_file="$dist/ainovel-cli_checksums.txt"
[ -s "$checksum_file" ] || { echo "missing checksum file: $checksum_file" >&2; exit 1; }

if command -v sha256sum >/dev/null 2>&1; then
  sha256_file() { sha256sum "$1" | awk '{ print tolower($1) }'; }
elif command -v shasum >/dev/null 2>&1; then
  sha256_file() { shasum -a 256 "$1" | awk '{ print tolower($1) }'; }
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi

expected="ainovel-cli_${version}_Darwin_arm64.tar.gz
ainovel-cli_${version}_Darwin_x86_64.tar.gz
ainovel-cli_${version}_Linux_arm64.tar.gz
ainovel-cli_${version}_Linux_x86_64.tar.gz
ainovel-cli_${version}_Windows_arm64.zip
ainovel-cli_${version}_Windows_x86_64.zip"

verify_tar() {
  asset="$1"
  entries_file="$(mktemp)"
  details_file="$(mktemp)"
  trap 'rm -f "$entries_file" "$details_file"' EXIT INT TERM
  tar -tzf "$dist/$asset" > "$entries_file"
  tar -tvzf "$dist/$asset" > "$details_file"
  awk '
    NF && ($0 == "ainovel-cli" || $0 == "README.md" || $0 == "LICENSE") { seen[$0]++; next }
    NF { exit 1 }
    END { exit !(seen["ainovel-cli"] == 1 && seen["README.md"] == 1 && seen["LICENSE"] == 1) }
  ' "$entries_file" || { echo "invalid or unsafe tar archive: $asset" >&2; exit 1; }
  awk 'NF && substr($1, 1, 1) != "-" { exit 1 }' "$details_file" || { echo "tar archive contains non-regular entry: $asset" >&2; exit 1; }
  rm -f "$entries_file" "$details_file"
  trap - EXIT INT TERM
}

verify_zip() {
  asset="$1"
  entries_file="$(mktemp)"
  trap 'rm -f "$entries_file"' EXIT INT TERM
  unzip -Z1 "$dist/$asset" > "$entries_file"
  awk '
    NF && ($0 == "ainovel-cli.exe" || $0 == "README.md" || $0 == "LICENSE") { seen[$0]++; next }
    NF { exit 1 }
    END { exit !(seen["ainovel-cli.exe"] == 1 && seen["README.md"] == 1 && seen["LICENSE"] == 1) }
  ' "$entries_file" || { echo "invalid or unsafe zip archive: $asset" >&2; exit 1; }
  rm -f "$entries_file"
  trap - EXIT INT TERM
}

printf '%s\n' "$expected" | while IFS= read -r asset; do
  [ -s "$dist/$asset" ] || { echo "missing artifact: $asset" >&2; exit 1; }
  count="$(awk -v asset="$asset" '$2 == asset || $2 == "*" asset { count++ } END { print count + 0 }' "$checksum_file")"
  [ "$count" -eq 1 ] || { echo "checksum entry is not unique: $asset" >&2; exit 1; }
  expected_sum="$(awk -v asset="$asset" '$2 == asset || $2 == "*" asset { print tolower($1) }' "$checksum_file")"
  actual_sum="$(sha256_file "$dist/$asset")"
  [ "$expected_sum" = "$actual_sum" ] || { echo "checksum mismatch: $asset" >&2; exit 1; }
  case "$asset" in
    *.tar.gz) verify_tar "$asset" ;;
    *.zip) verify_zip "$asset" ;;
  esac
done

extra="$(find "$dist" -maxdepth 1 -type f -name 'ainovel-cli_*' -print | sed 's#^.*/##' | sort | grep -v -E "^($(printf '%s\n' "$expected" | tr '\n' '|' | sed 's/|$//'))$|^ainovel-cli_checksums\.txt$" || true)"
[ -z "$extra" ] || { echo "unexpected release artifacts:" >&2; printf '%s\n' "$extra" >&2; exit 1; }
