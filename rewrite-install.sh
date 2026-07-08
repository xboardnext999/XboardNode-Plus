#!/usr/bin/env bash
set -euo pipefail

OLD_URL="https://raw.githubusercontent.com/cedar2025/xboard-node/dev/install.sh"
NEW_URL="https://raw.githubusercontent.com/xboardnext999/XboardNode-Plus/dev/install.sh"
REPO_URL="https://github.com/xboardnext999/XboardNode-Plus.git"
REPO_BRANCH="dev"
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

If raw.githubusercontent.com is rate-limited, the script will fall back to
cloning ${REPO_URL} and running install.sh locally with the same arguments.
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

extract_installer_args() {
    local cmd="$1"
    case "$cmd" in
        *"bash -s -- "*)
            printf '%s\n' "${cmd#*bash -s -- }"
            ;;
        *"bash -s --")
            printf '\n'
            ;;
        *)
            return 1
            ;;
    esac
}

fallback_install_from_git() {
    local cmd="$1"
    local args_str
    if ! args_str="$(extract_installer_args "$cmd")"; then
        return 1
    fi

    local tmp_dir status runner
    tmp_dir="$(mktemp -d)"
    say "直接下载安装脚本失败，尝试使用 git clone 方式安装..."
    if ! git clone --depth 1 -b "$REPO_BRANCH" "$REPO_URL" "$tmp_dir"; then
        rm -rf "$tmp_dir"
        return 1
    fi

    runner="bash"
    if [[ "$cmd" == *"| sudo bash"* ]] || [[ "$cmd" == *"sudo bash -s"* ]]; then
        runner="sudo bash"
    fi

    set +e
    bash -lc "$runner $(printf '%q' "$tmp_dir/install.sh") $args_str"
    status=$?
    set -e
    rm -rf "$tmp_dir"
    return "$status"
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
say "提示：上面的 token 仅为显示时脱敏，实际执行会使用你粘贴命令中的原始 token。"

if [ "$PRINT_ONLY" -eq 1 ]; then
    printf '%s\n' "$converted_cmd"
    exit 0
fi

set +e
bash -o pipefail -lc "$converted_cmd"
status=$?
set -e
if [ "$status" -ne 0 ]; then
    if fallback_install_from_git "$converted_cmd"; then
        exit 0
    fi
    exit "$status"
fi
