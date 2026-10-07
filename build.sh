#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
output_dir="${project_dir}/build"
output_file="${output_dir}/note-vault"
config_file="${output_dir}/config.yaml"
frontend_source_dir="${project_dir}/dist"
frontend_output_dir="${output_dir}/dist"
go_bin="/data/program/go1.27.1.linux-amd64/bin/go"

if [ ! -x "${go_bin}" ]; then
  go_bin="$(command -v go)"
fi

mkdir -p "${output_dir}"

cd "${project_dir}"

CGO_ENABLED=0 \
GOOS=linux \
GOARCH=amd64 \
"${go_bin}" build \
  -trimpath \
  -ldflags="-s -w" \
  -o "${output_file}" \
  .

if [ ! -f "${config_file}" ]; then
  cp "${project_dir}/config.yaml" "${config_file}"
else
  echo "保留已有配置文件：${config_file}"
fi

if [ -d "${frontend_source_dir}" ]; then
  rm -rf "${frontend_output_dir}"
  cp -a "${frontend_source_dir}" "${frontend_output_dir}"
  echo "前端静态资源：${frontend_output_dir}"
else
  echo "警告：未找到前端构建目录：${frontend_source_dir}" >&2
fi

echo "构建完成：${output_file}"
echo "配置文件：${config_file}"
echo "启动方式：cd ${output_dir} && ./note-vault"
