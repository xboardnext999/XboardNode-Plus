#!/usr/bin/env bash
set -euo pipefail

OLD_URL="https://raw.githubusercontent.com/cedar2025/xboard-node/dev/install.sh"
NEW_URL="https://raw.githubusercontent.com/xboardnext999/XboardNode-Plus/dev/install.sh"
PRINT_ONLY=0

usage() {
    cat <<EOF
Usage:
  rewrite-install.sh [--print-only] ["original install command"]

Interactive:
  curl -fsSL ${NEW_URL%/install.sh}/rewrite-install.sh | bash

Then paste the original cedar2025 install command. The script will replace:
  ${OLD_URL}
with:
  ${NEW_URL}
and execute the converted command.
EOF
}

say() {
    if { printf '%s\n' "$*" > /dev/tty; } 2>/dev/null; then
        return
    fi
    printf '%s\n' "$*"
}

die() {
    say "错误: $*"
    exit 1
}

mask_token() {
    sed -E \
        -e "s/(--token[[:space:]]+)'[^']*'/\1'***'/g" \
        -e 's/(--token[[:space:]]+)"[^"]*"/\1"***"/g' \
        -e 's/(--token[[:space:]]+)[^[:space:]]+/\1***/g'
}

read_command() {
    if [ "$#" -gt 0 ]; then
        printf '%s\n' "$*"
        return
    fi

    if { exec 3< /dev/tty; } 2>/dev/null; then
        say "请粘贴原作者安装命令，然后按 Enter："
        IFS= read -r line <&3 || true
        exec 3<&-
    else
        IFS= read -r line || true
    fi
    printf '%s\n' "${line:-}"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --print-only|-p)
            PRINT_ONLY=1
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            break
            ;;
    esac
done

original_cmd="$(read_command "$@")"
if [ -z "${original_cmd//[[:space:]]/}" ]; then
    die "没有读取到安装命令"
fi

if [[ "$original_cmd" == *"$OLD_URL"* ]]; then
    converted_cmd="${original_cmd//$OLD_URL/$NEW_URL}"
elif [[ "$original_cmd" == *"$NEW_URL"* ]]; then
    converted_cmd="$original_cmd"
else
    die "没有找到原作者安装地址：${OLD_URL}"
fi

masked_cmd="$(printf '%s' "$converted_cmd" | mask_token)"
say "将执行替换后的命令："
say "$masked_cmd"

if [ "$PRINT_ONLY" -eq 1 ]; then
    printf '%s\n' "$converted_cmd"
    exit 0
fi

bash -lc "$converted_cmd"
