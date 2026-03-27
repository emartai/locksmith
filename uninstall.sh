#!/bin/sh

set -eu

remove_if_present() {
  path="$1"
  if [ -f "$path" ]; then
    rm -f "$path"
    echo "removed $path"
    return 0
  fi
  return 1
}

if remove_if_present "/usr/local/bin/locksmith"; then
  exit 0
fi

if remove_if_present "$HOME/.local/bin/locksmith"; then
  exit 0
fi

echo "locksmith is not installed in /usr/local/bin or \$HOME/.local/bin" >&2
exit 1
