#! /bin/sh

set -eux

go get -u github.com/gobuffalo/packr/packr
go get -u github.com/ribice/glice
go get -u github.com/kissen/fmajor

current_diretory="$(pwd)"

source_directory="$GOPATH/src/github.com/kissen/fmajor"
license_directory="$source_directory/licenses"

temp_directory=$(mktemp -d)
release_directory="$temp_directory/fmajor"

# cd into $directory which is the root of our operations
cd "$temp_directory"

# in the source directory, build binary and license files
(
    cd "$source_directory"

    packr build
    glice -s -r -f
)

# in the release directory, assemble all required files

(
    mkdir -p "$release_directory"
    cd "$release_directory"

    cp "$source_directory/fmajor" .
    cp "$source_directory/README.md" .

    mkdir doc
    cp "$source_directory/doc/fmajor.conf" doc/
    cp "$source_directory/doc/fmajor.service" doc/

    mkdir licenses
    cp -r "$license_directory" "licenses"
)

# now to finish up create the archive

tar cfz fmajor.tar.gz "fmajor"
cp "$temp_directory/fmajor.tar.gz" "$current_diretory/fmajor.tar.gz"
