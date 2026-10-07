#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
app_path="${project_dir}/build/note-vault"
app_dir="${project_dir}/build"
pid_file="${app_dir}/note-vault.pid"
log_file="${app_dir}/note-vault.log"

print_usage() {
  echo "用法：$0 {start|status|stop}"
}

read_pid() {
  if [ ! -f "${pid_file}" ]; then
    return 1
  fi

  cat "${pid_file}"
}

is_running() {
  local app_pid

  app_pid="$(read_pid 2>/dev/null || true)"
  if [ -z "${app_pid}" ] || ! [[ "${app_pid}" =~ ^[0-9]+$ ]]; then
    return 1
  fi

  if ! kill -0 "${app_pid}" 2>/dev/null; then
    return 1
  fi

  if [ -r "/proc/${app_pid}/cmdline" ]; then
    tr '\0' ' ' < "/proc/${app_pid}/cmdline" | grep -Fq -- "${app_path}"
    return $?
  fi

  return 0
}

remove_stale_pid_file() {
  if [ -f "${pid_file}" ] && ! is_running; then
    rm -f "${pid_file}"
  fi
}

start_service() {
  if [ ! -x "${app_path}" ]; then
    echo "错误：找不到可执行文件或没有执行权限：${app_path}" >&2
    echo "请先执行 ./build.sh" >&2
    exit 1
  fi

  if [ ! -f "${app_dir}/config.yaml" ]; then
    echo "错误：找不到配置文件：${app_dir}/config.yaml" >&2
    exit 1
  fi

  remove_stale_pid_file

  if is_running; then
    echo "服务已经在运行，PID=$(read_pid)"
    return 0
  fi

  mkdir -p "${app_dir}"

  (
    cd "${app_dir}"
    exec "${app_path}" >> "${log_file}" 2>&1
  ) &
  echo $! > "${pid_file}"

  sleep 1
  if is_running; then
    echo "服务启动成功，PID=$(read_pid)"
    echo "日志文件：${log_file}"
    return 0
  fi

  echo "服务启动失败，请查看日志：${log_file}" >&2
  rm -f "${pid_file}"
  exit 1
}

status_service() {
  remove_stale_pid_file

  if is_running; then
    echo "服务正在运行，PID=$(read_pid)"
    echo "日志文件：${log_file}"
    return 0
  fi

  echo "服务未运行"
  return 1
}

stop_service() {
  remove_stale_pid_file

  if ! is_running; then
    echo "服务未运行"
    rm -f "${pid_file}"
    return 0
  fi

  local app_pid
  app_pid="$(read_pid)"
  kill "${app_pid}"

  for _ in $(seq 1 30); do
    if ! is_running; then
      rm -f "${pid_file}"
      echo "服务已停止"
      return 0
    fi
    sleep 1
  done

  echo "服务未能在 30 秒内停止，执行强制停止：PID=${app_pid}" >&2
  kill -9 "${app_pid}" 2>/dev/null || true
  rm -f "${pid_file}"
  echo "服务已强制停止"
}

case "${1:-}" in
  start)
    start_service
    ;;
  status)
    status_service
    ;;
  stop)
    stop_service
    ;;
  *)
    print_usage
    exit 1
    ;;
esac
