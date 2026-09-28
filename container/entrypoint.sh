#!/bin/sh
set -e

case "$1" in
    tui)
        shift
        exec /usr/local/bin/goweeb-tui "$@"
        ;;

    cli)
        shift
        exec /usr/local/bin/goweeb-cli "$@"
        ;;

    *)
        echo "Unknown mode: $1" >&2
        echo "Usage: goweeb [tui|cli] [arguments...]" >&2
        exit 1
        ;;
esac
