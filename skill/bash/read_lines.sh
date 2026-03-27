#!/bin/bash
# 文件名: read_lines.sh

set -euo pipefail

# 显示帮助信息
show_help() {
    cat << EOF
用法: $0 文件名 [起始行] [读取行数]

功能: 从指定文件读取特定行范围

参数:
  文件名      - 必需，要读取的文件路径
  起始行      - 可选，从第几行开始读取（默认: 1）
  读取行数    - 可选，读取多少行（默认: 到文件末尾）

示例:
  $0 file.txt              # 读取整个文件
  $0 file.txt 10           # 从第10行读到末尾
  $0 file.txt 5 3          # 从第5行开始，读3行
  $0 file.txt 2 0          # 只读取第2行
  $0 -                     # 从标准输入读取
  $0 file.txt tail 5       # 读取最后5行
  $0 file.txt head 5       # 读取前5行

EOF
    exit 0
}

#参数检查
if [[ $# -eq 0 ]] || [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
  show_help
fi

FILE="$1"
TMP_FILE=""
if [[ "$FILE" == "-" ]]; then
    TMP_FILE=$(mktemp)
    trap 'rm -f "$TMP_FILE"' EXIT
    cat > "$TMP_FILE"
    FILE="$TMP_FILE"
fi

if [[ -z "$TMP_FILE" ]] && [[ ! -f "$FILE" ]]; then
    echo "文件不存在"
    exit 1
fi
TOTAL_LINES=$(wc -l < "$FILE")
START_LINE=1
LINE_COUNT=0

case $# in
  1)
    cat "$FILE"
    exit 0
    ;;
  2)
    if ! [[ "$2" =~ ^[0-9]+$ ]]; then
      echo "错误：起始行必须是数字" >&2
      exit 1
    fi
    START_LINE="$2"
    if [[ "$START_LINE" -lt 1 ]]; then
      echo "错误：起始行不能小于1" >&2
      exit 1
    fi
    ;;
  3)
    if [[ "$2" == "tail" ]]; then
        if ! [[ "$3" =~ ^[0-9]+$ ]]; then
          echo "错误: tail 的行数必须是数字" >&2
          exit 1
        fi
        tail -n "$3" "$FILE"
        exit 0
    elif [[ "$2" == "head" ]]; then
      if ! [[ "$3" =~ ^[0-9]+$ ]]; then
        echo "错误: head 的行数必须是数字" >&2
        exit 1
      fi
      head -n "$3" "$FILE"
      exit 0
    else
      if ! [[ "$2"  =~ ^[0-9]+$ ]]; then
        echo "错误：起始行必须是数字" >&2
        exit 1
      fi
      if ! [[ "$3" =~ ^[0-9]+$ ]]; then
        echo "错误：行数必须是数字" >&2
        exit 1
      fi
      START_LINE="$2"
      LINE_COUNT="$3"
    fi
    ;;
  *)
    show_help
    ;;
esac

# 检查起始行是否超出范围
if [[ "$START_LINE" -gt "$TOTAL_LINES" ]]; then
    echo "警告: 起始行($START_LINE)超过文件总行数($TOTAL_LINES)" >&2
    exit 0
fi

# 计算结束行
if [[ "$LINE_COUNT" -eq 0 ]]; then
    sed -n "${START_LINE},\$p" "$FILE"
else
    END_LINE=$((START_LINE + LINE_COUNT - 1))
    if [[ "$END_LINE" -gt "$TOTAL_LINES" ]]; then
        END_LINE="$TOTAL_LINES"
    fi
    sed -n "${START_LINE},${END_LINE}p" "$FILE"
fi


