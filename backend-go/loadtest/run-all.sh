#!/usr/bin/env bash
# 顺序运行 5 个 k6 压测场景并汇总结果
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RESULTS_DIR="$SCRIPT_DIR/results"
mkdir -p "$RESULTS_DIR"

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
SUMMARY_FILE="$RESULTS_DIR/summary-$TIMESTAMP.md"

SCENARIOS=(
  "dashboard|dashboard.js"
  "ai-command|ai-command.js"
  "action-approve|action-approve.js"
  "websocket|websocket.js"
  "sku-batch|sku-batch.js"
)

API_BASE="${API_BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-}"

echo "=========================================="
echo " k6 压测 - 全场景顺序执行"
echo " API_BASE : $API_BASE"
echo " TIMESTAMP: $TIMESTAMP"
echo "=========================================="
echo ""

# 汇总表头
{
  echo "# k6 压测汇总 ($TIMESTAMP)"
  echo ""
  echo "- API_BASE: \`$API_BASE\`"
  echo "- TOKEN: ${TOKEN:+已设置}${TOKEN:-未设置}"
  echo ""
  echo "| 场景 | 耗时 | http_reqs | p95(ms) | http_req_failed | checks通过率 | 状态 |"
  echo "|------|------|-----------|---------|-----------------|--------------|------|"
} > "$SUMMARY_FILE"

overall_ok=0

for entry in "${SCENARIOS[@]}";
do
  name="${entry%%|*}"
  script="${entry##*|}"

  result_file="$RESULTS_DIR/${name}-${TIMESTAMP}.json"
  log_file="$RESULTS_DIR/${name}-${TIMESTAMP}.log"
  summary_json="$RESULTS_DIR/${name}-${TIMESTAMP}-summary.json"

  echo "------------------------------------------"
  echo "[$(date +%H:%M:%S)] 场景: $name ($script)"
  echo "------------------------------------------"

  if [[ ! -f "$script" ]]; then
    echo "[SKIP] 脚本不存在: $script"
    echo "| $name | - | - | - | - | - | SKIP(脚本缺失) |" >> "$SUMMARY_FILE"
    continue
  fi

  start_ts=$(date +%s)

  # 运行 k6，输出 JSON + 终端日志
  set +e
  k6 run \
    --quiet \
    --out "json=$result_file" \
    --summary-export "$summary_json" \
    "$script" 2>&1 | tee "$log_file"
  k6_exit=${PIPESTATUS[0]}
  set -e

  end_ts=$(date +%s)
  duration=$((end_ts - start_ts))

  # 从 summary-export 的 JSON 结果中提取指标
  if [[ -f "$summary_json" ]]; then
    read http_reqs p95 fail_rate checks_pass <<< $(python3 -c "
import json
try:
    with open('$summary_json') as f:
        d = json.load(f)
    m = d.get('metrics', {})
    reqs = m.get('http_reqs', {}).get('count', 0)
    dur = m.get('http_req_duration', {}).get('p(95)', 0.0)
    if '$name' == 'websocket':
        p95_str = 'n/a'
    else:
        p95_str = f'{dur:.1f}ms'
    failed_val = m.get('http_req_failed', {}).get('value', 0.0)
    fail = f'{failed_val * 100:.2f}%' if 'http_req_failed' in m else 'n/a'

    checks_m = m.get('checks', {})
    if 'value' in checks_m:
        chk = f'{checks_m[\"value\"] * 100:.2f}%'
    else:
        chk = '100.00%'
    print(f'{reqs} {p95_str} {fail} {chk}')
except Exception as e:
    print('0 n/a n/a n/a')
")
  else
    http_reqs="0"
    p95="n/a"
    fail_rate="n/a"
    checks_pass="n/a"
  fi

  if [[ $k6_exit -eq 0 ]]; then
    status="PASS"
  else
    status="FAIL(exit=$k6_exit)"
    overall_ok=1
  fi

  echo ""
  echo "  耗时: ${duration}s  http_reqs: $http_reqs  p95: ${p95}  failed: $fail_rate  checks: $checks_pass  -> $status"
  echo ""

  echo "| $name | ${duration}s | $http_reqs | $p95 | $fail_rate | $checks_pass | $status |" >> "$SUMMARY_FILE"
done

echo "=========================================="
echo " 全部场景执行完成"
echo " 汇总报告: $SUMMARY_FILE"
echo " 结果目录: $RESULTS_DIR"
echo "=========================================="

echo ""
echo "---------- 汇总 ----------"
cat "$SUMMARY_FILE"

exit $overall_ok
