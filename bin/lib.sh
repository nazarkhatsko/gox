# Output helpers shared by the scripts in bin and .githooks; source, don't run.

if [ -t 1 ]; then
	green=$(printf '\033[32m') red=$(printf '\033[31m') dim=$(printf '\033[2m') reset=$(printf '\033[0m')
else
	green='' red='' dim='' reset=''
fi

fail() {
	echo "${red}✗${reset} $*" >&2
	exit 1
}

# step runs a command quietly and prints its output only if it fails.
step() {
	label=$1
	shift

	printf '%-14s' "$label"
	if out=$("$@" 2>&1); then
		echo "${green}✓${reset}"
	else
		echo "${red}✗${reset}"
		echo "$out" >&2
		exit 1
	fi
}

# require_clean fails when the working tree has changes, so checks run on
# exactly what is committed.
require_clean() {
	[ -z "$(git status --porcelain)" ] || fail "commit or stash your changes first"
}
