#!/bin/sh

set -eu

if ! command -v lefthook >/dev/null 2>&1; then
	echo "lefthook is not installed."
	echo "Install it first, then rerun this script."
	echo "Example: brew install lefthook"
	exit 1
fi

current_hooks_path="$(git config --local --get core.hooksPath || true)"
if [ "$current_hooks_path" = ".githooks" ]; then
	git config --local --unset core.hooksPath
fi

lefthook install
echo "Installed Lefthook hooks"
