#! /bin/bash

set -eu

if [ ! $# -eq 2 ]; then
	echo "usage: $0 HOST FILE" 1>&2
	exit 1
fi

# the address under which an fmajor instance is running
fmajor_host="$1"

# filepath is where the file to upload should be at
filepath="$2"
filename="$(basename "$filepath")"
filedir="$(dirname "$filepath")"

# ensure that the file exists (well at least right now)
if [ ! -f "$filepath" ]; then
    echo "$0: $filepath: no such file or directory" 1>&2
    exit 1
fi

# the password used for authentication
read -r -s -p "Password: " password

(
	cd "$filedir"
	curl -s -u "nobody:$password" -F "file=@$filename" "$fmajor_host"/submit
)
