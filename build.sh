#!/usr/bin/env sh

gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -os="linux windows"
