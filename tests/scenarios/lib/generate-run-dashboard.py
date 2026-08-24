#!/usr/bin/env python3
"""Generate a self-contained Chart.js dashboard for one MTV benchmark run.

Usage:
  generate-run-dashboard.py <artifact-prefix>

Looks for:
  <prefix>-kc-mem/{vm1,vm2,vm3}-virt-v2v-memory.csv
  <prefix>-ref-mem/{vm1,vm2,vm3}-virt-v2v-memory.csv
  (also accepts legacy rhel/win CSV names)
  <prefix>-{kc,ref}.log  (optional; VM walls, migration start/end, plan duration)

Writes:
  <prefix>.html
"""

from __future__ import annotations

import csv
import json
import re
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

CHART_JS_PATH = Path(__file__).resolve().parent / "chart.umd.min.js"


def load_chart_js() -> str:
    if not CHART_JS_PATH.is_file():
        raise FileNotFoundError(
            f"missing {CHART_JS_PATH} (Chart.js bundle for dashboards)"
        )
    return CHART_JS_PATH.read_text()


def load_csv(path: Path) -> list[dict[str, Any]]:
    if not path.is_file():
        return []
    rows: list[dict[str, Any]] = []
    prev_t: int | None = None
    prev_rx_bytes: int | None = None
    with path.open(newline="") as f:
        for row in csv.DictReader(f):
            try:
                t = int(float(row.get("elapsed_s") or 0))
            except ValueError:
                continue
            mem_raw = row.get("mem_rss_mi_cgroup") or row.get("mem_working_set_mi") or ""
            cpu_raw = row.get("cpu_m") or ""
            rx_raw = row.get("net_rx_bytes") or ""
            try:
                mem = int(float(mem_raw)) if mem_raw != "" else None
            except ValueError:
                mem = None
            try:
                cpu = int(float(cpu_raw)) if cpu_raw != "" else None
            except ValueError:
                cpu = None
            try:
                rx_bytes = int(float(rx_raw)) if rx_raw != "" else None
            except ValueError:
                rx_bytes = None
            rx = int(rx_bytes / 1024 / 1024) if rx_bytes is not None else None
            rx_rate = None
            if (
                rx_bytes is not None
                and prev_rx_bytes is not None
                and prev_t is not None
                and t > prev_t
                and rx_bytes >= prev_rx_bytes
            ):
                delta_mi = (rx_bytes - prev_rx_bytes) / 1024 / 1024
                rx_rate = round(delta_mi / (t - prev_t), 2)
            if rx_bytes is not None:
                prev_t = t
                prev_rx_bytes = rx_bytes
            rows.append({"t": t, "mem": mem, "cpu": cpu, "rx": rx, "rx_rate": rx_rate})
    return rows


def peak(samples: list[dict[str, Any]], key: str) -> int | float | None:
    vals = [s[key] for s in samples if s.get(key) is not None]
    return max(vals) if vals else None


def last(samples: list[dict[str, Any]], key: str) -> int | None:
    for s in reversed(samples):
        if s.get(key) is not None:
            return s[key]
    return None


CSV_SUFFIX = "-virt-v2v-memory.csv"
PREFERRED_VM_LABELS = ("vm1", "vm2", "vm3", "rhel", "win")


def csv_vm_label(path: Path) -> str | None:
    name = path.name
    if name.endswith(CSV_SUFFIX):
        return name[: -len(CSV_SUFFIX)]
    return None


def discover_vm_labels(mem_dirs: list[Path]) -> list[str]:
    found: set[str] = set()
    for mem_dir in mem_dirs:
        for path in mem_dir.glob(f"*{CSV_SUFFIX}"):
            label = csv_vm_label(path)
            if label:
                found.add(label)
    preferred = [label for label in PREFERRED_VM_LABELS if label in found]
    rest = sorted(found.difference(PREFERRED_VM_LABELS))
    return preferred + rest


def js_ident(label: str) -> str:
    """Sanitize a VM label into a JavaScript identifier (const / call args)."""
    ident = re.sub(r"[^A-Za-z0-9_]", "_", label.upper())
    if not ident or ident[0].isdigit():
        ident = f"_{ident}"
    return ident


def vm_display(label: str) -> str:
    return {"rhel": "RHEL", "win": "Windows"}.get(label, label.upper())


def parse_log_meta(log_path: Path, vm_labels: list[str]) -> dict[str, str]:
    meta: dict[str, str] = {}
    if not log_path.is_file():
        return meta
    text = log_path.read_text(errors="replace")
    m = re.search(r"virt_v2v_image_fqin:\s*(\S+)", text)
    if m:
        meta["image"] = m.group(1)
    for label in vm_labels:
        upper = label.upper()
        name_m = re.search(rf"^{re.escape(upper)}=(\S+)", text, re.M)
        if not name_m and label == "rhel":
            name_m = re.search(r"^RHEL_VM=(\S+)", text, re.M)
        if not name_m and label == "win":
            name_m = re.search(r"^WIN_VM=(\S+)", text, re.M)
        if name_m:
            meta[f"{label}_vm"] = name_m.group(1)
        m = re.search(
            rf"RESULT {re.escape(label)}: status=(\S+) total=(\S+)",
            text,
        )
        if m:
            meta[f"{label}_status"] = m.group(1)
            meta[f"{label}_wall"] = m.group(2)
    m = re.search(r"Migration started at (\S+)", text)
    if m:
        meta["started_at"] = m.group(1)
    m = re.search(r"Migration ended at (\S+)", text)
    if m:
        meta["ended_at"] = m.group(1)
    m = re.search(r"Migration lifetime: (\S+)", text)
    if m:
        meta["lifetime"] = m.group(1)
    else:
        m = re.search(r"Plan \S+ finished: status=\S+ duration=(\S+)", text)
        if m:
            meta["lifetime"] = m.group(1)
    if "ended_at" not in meta:
        ended = add_wall(meta.get("started_at"), meta.get("lifetime"))
        if ended:
            meta["ended_at"] = ended
    return meta


def fmt_peak(v: int | float | None, unit: str) -> str:
    if v is None:
        return "—"
    if isinstance(v, float):
        return f"{v:.1f} {unit}"
    return f"{v} {unit}"


def parse_iso_utc(value: str | None) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def fmt_iso_utc(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def add_wall(iso: str | None, wall: str | None) -> str | None:
    dt = parse_iso_utc(iso)
    secs = parse_wall_s(wall)
    if dt is None or secs is None:
        return None
    return fmt_iso_utc(dt + timedelta(seconds=secs))


def parse_wall_s(wall: str | None) -> int | None:
    """Parse fmt_dur output like '12m51s' into seconds."""
    if not wall:
        return None
    m = re.fullmatch(r"(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?", wall)
    if not m or not any(m.groups()):
        return None
    hours = int(m.group(1) or 0)
    mins = int(m.group(2) or 0)
    secs = int(m.group(3) or 0)
    return hours * 3600 + mins * 60 + secs


def fmt_dur_s(seconds: int) -> str:
    sign = "-" if seconds < 0 else ""
    s = abs(seconds)
    if s < 60:
        return f"{sign}{s}s"
    return f"{sign}{s // 60}m{s % 60:02d}s"


def fmt_dur_diff(newer_s: int | None, baseline_s: int | None) -> str:
    """Format newer − baseline (negative = newer faster)."""
    if newer_s is None or baseline_s is None or baseline_s == 0:
        return "—"
    delta = newer_s - baseline_s
    pct = 100.0 * delta / baseline_s
    return f"{fmt_dur_s(delta)} ({pct:+.1f}%)"


def series_js(name: str, samples: list[dict[str, Any]]) -> str:
    return f"const {name}={json.dumps(samples, separators=(',', ':'))};"


def build_html(
    title: str,
    meta_line: str,
    converters: list[str],
    data: dict[str, dict[str, list[dict[str, Any]]]],
    meta: dict[str, dict[str, str]],
    vm_labels: list[str],
) -> str:
    """converters: ordered list like ['ref','kc'] or ['kc'].
    data[converter][vm] = samples
    """
    # Prefer compare order: ref then kc when both present
    order = [c for c in ("ref", "kc") if c in converters] or converters
    labels = {"ref": "ref (virt-v2v)", "kc": "kc-v2v"}
    colors = {
        "ref": {"mem": "#868e96", "cpu": "#1971c2", "rx": "#868e96"},
        "kc": {"mem": "#2b8a3e", "cpu": "#e67700", "rx": "#2b8a3e"},
    }

    # Embed series consts
    consts: list[str] = []
    for conv in order:
        for vm in vm_labels:
            key = f"{js_ident(vm)}_{conv.upper()}"
            consts.append(series_js(key, data.get(conv, {}).get(vm, [])))

    # Duration / peak / net table rows
    show_dur_diff = "ref" in order and "kc" in order
    dur_rows = []
    peak_rows = []
    net_rows = []
    for vm in vm_labels:
        vm_title = vm_display(vm)
        dur_cells = [f"<td>{vm_title}</td>"]
        peak_cells = [f"<td>{vm_title}</td>"]
        net_cells = [f"<td>{vm_title}</td>"]
        walls_s: dict[str, int | None] = {}
        for conv in order:
            m = meta.get(conv, {})
            wall = m.get(f"{vm}_wall")
            status = m.get(f"{vm}_status", "?")
            walls_s[conv] = parse_wall_s(wall)
            if wall:
                dur_cells.append(f"<td>{wall} ({status})</td>")
            else:
                dur_cells.append("<td>—</td>")
            samples = data.get(conv, {}).get(vm, [])
            peak_cells.append(f"<td>{fmt_peak(peak(samples, 'mem'), 'Mi')}</td>")
            peak_cells.append(f"<td>{fmt_peak(peak(samples, 'cpu'), 'm')}</td>")
            net_cells.append(f"<td>{fmt_peak(last(samples, 'rx'), 'Mi')}</td>")
            net_cells.append(f"<td>{fmt_peak(peak(samples, 'rx_rate'), 'Mi/s')}</td>")
        if show_dur_diff:
            dur_cells.append(
                f"<td>{fmt_dur_diff(walls_s.get('kc'), walls_s.get('ref'))}</td>"
            )
        dur_rows.append("<tr>" + "".join(dur_cells) + "</tr>")
        peak_rows.append("<tr>" + "".join(peak_cells) + "</tr>")
        net_rows.append("<tr>" + "".join(net_cells) + "</tr>")

    dur_head = "".join(f"<th>{labels.get(c, c)}</th>" for c in order)
    if show_dur_diff:
        dur_head += "<th>Δ (kc − ref)</th>"
    peak_head = "".join(
        f"<th>{labels.get(c, c)} peak mem</th><th>{labels.get(c, c)} peak CPU</th>"
        for c in order
    )
    net_head = "".join(
        f"<th>{labels.get(c, c)} RX</th><th>{labels.get(c, c)} peak Mi/s</th>"
        for c in order
    )

    has_duration = any(
        meta.get(c, {}).get(f"{vm}_wall")
        for c in order
        for vm in vm_labels
    )
    duration_section = ""
    if has_duration:
        duration_section = f"""
<h2>Duration</h2>
<table>
<tr><th>VM</th>{dur_head}</tr>
{"".join(dur_rows)}
</table>
"""

    def meta_cell(conv: str, key: str) -> str:
        value = meta.get(conv, {}).get(key)
        return f"<td>{value}</td>" if value else "<td>—</td>"

    has_lifetime = any(
        meta.get(c, {}).get("lifetime") or meta.get(c, {}).get("started_at")
        for c in order
    )
    lifetime_section = ""
    if has_lifetime:
        start_cells = ["<td>Start (UTC)</td>"]
        end_cells = ["<td>End (UTC)</td>"]
        life_cells = ["<td>Lifetime</td>"]
        life_s: dict[str, int | None] = {}
        for conv in order:
            start_cells.append(meta_cell(conv, "started_at"))
            end_cells.append(meta_cell(conv, "ended_at"))
            life = meta.get(conv, {}).get("lifetime")
            life_s[conv] = parse_wall_s(life)
            life_cells.append(f"<td>{life}</td>" if life else "<td>—</td>")
        if show_dur_diff:
            start_cells.append("<td></td>")
            end_cells.append("<td></td>")
            life_cells.append(
                f"<td>{fmt_dur_diff(life_s.get('kc'), life_s.get('ref'))}</td>"
            )
        lifetime_section = f"""
<h2>Migration lifetime</h2>
<p class="caption">From <code>oc mtv start</code> until the plan is terminal
(log lines: <code>Migration started at</code>, <code>Migration ended at</code>,
or <code>Plan … finished … duration=</code>).</p>
<table>
<tr><th></th>{dur_head}</tr>
<tr>{"".join(start_cells)}</tr>
<tr>{"".join(end_cells)}</tr>
<tr>{"".join(life_cells)}</tr>
</table>
"""
    series_args = {
        vm: ", ".join(f"{js_ident(vm)}_{c.upper()}" for c in order) for vm in vm_labels
    }
    legend_names = [labels.get(c, c) for c in order]
    color_mem = [colors[c]["mem"] for c in order]
    color_cpu = [colors[c]["cpu"] for c in order]
    color_rx = [colors[c]["rx"] for c in order]

    chart_calls = []
    mem_sections = []
    net_sections = []
    for i, vm in enumerate(vm_labels):
        title_vm = vm_display(vm)
        caption = (
            '<p class="caption">x-axis = elapsed seconds since monitor start</p>\n'
            if i == 0
            else ""
        )
        mem_sections.append(
            f"<h2>{title_vm} conversion pod — memory &amp; CPU over time</h2>\n"
            f'<div class="chart-box"><canvas id="{vm}MemCpu"></canvas></div>\n'
            f"{caption}"
        )
        net_cap = (
            '<p class="caption">Solid = cumulative RX from /proc/net/dev '
            "(all non-lo interfaces). Dashed = Mi/s vs the previous sample "
            "(Δbytes / Δelapsed).</p>\n"
            if i == 0
            else ""
        )
        net_sections.append(
            f"<h2>{title_vm} conversion pod — network RX over time</h2>\n"
            f'<div class="chart-box"><canvas id="{vm}Net"></canvas></div>\n'
            f"{net_cap}"
        )
        chart_calls.append(f"mkMemCpu('{vm}MemCpu', {series_args[vm]});")
        chart_calls.append(f"mkNet('{vm}Net', {series_args[vm]});")

    app_js = f"""
const SERIES_LABELS = {json.dumps(legend_names)};
const COLOR_MEM = {json.dumps(color_mem)};
const COLOR_CPU = {json.dumps(color_cpu)};
const COLOR_RX = {json.dumps(color_rx)};

{chr(10).join(consts)}

function nearest(samples,t,key,win=35){{
  if(!samples||!samples.length)return NaN;
  let best=null;
  for(const s of samples){{const v=s[key];if(v==null)continue;const d=Math.abs(s.t-t);
    if(d<=win&&(best==null||d<best.d))best={{d,v}};}}
  return best?best.v:NaN;
}}
function overlay(seriesList,key,maxPts=22){{
  const nonempty=seriesList.filter(s=>s&&s.length);
  if(!nonempty.length)return{{labels:[],data:seriesList.map(()=>[])}};
  const tmax=Math.max(...nonempty.map(s=>s[s.length-1].t));
  const step=Math.max(30,Math.ceil(tmax/maxPts));
  const labels=[], data=seriesList.map(()=>[]);
  for(let t=0;t<=tmax;t+=step){{
    const m=Math.floor(t/60),s=t%60;
    labels.push(s===0?m+'m':m+':'+String(s).padStart(2,'0'));
    seriesList.forEach((samples,i)=>data[i].push(nearest(samples,t,key)));
  }}
  return{{labels,data}};
}}

function mkMemCpu(canvasId,...series){{
  const mem=overlay(series,'mem');
  const cpu=overlay(series,'cpu');
  const datasets=[];
  series.forEach((_,i)=>{{
    datasets.push({{label:SERIES_LABELS[i]+' mem (Mi)',data:mem.data[i],borderColor:COLOR_MEM[i],backgroundColor:COLOR_MEM[i]+'1a',tension:.3,pointRadius:1}});
    datasets.push({{label:SERIES_LABELS[i]+' cpu (m)',data:cpu.data[i],borderColor:COLOR_CPU[i],backgroundColor:COLOR_CPU[i]+'1a',tension:.3,pointRadius:1,borderDash:[4,3]}});
  }});
  new Chart(document.getElementById(canvasId),{{type:'line',
    data:{{labels:mem.labels,datasets}},
    options:{{responsive:true,maintainAspectRatio:false,
      scales:{{y:{{beginAtZero:true,title:{{display:true,text:'Mi / millicores'}}}},
              x:{{title:{{display:true,text:'Elapsed'}}}}}},
      plugins:{{legend:{{position:'bottom'}}}},interaction:{{mode:'index',intersect:false}}}}
  }});
}}
function overlayAll(seriesList,key){{
  const nonempty=seriesList.filter(s=>s&&s.length);
  if(!nonempty.length)return{{labels:[],data:seriesList.map(()=>[])}};
  const times=[...new Set(nonempty.flatMap(s=>s.map(p=>p.t)))].sort((a,b)=>a-b);
  const labels=[], data=seriesList.map(()=>[]);
  for(const t of times){{
    const m=Math.floor(t/60),s=t%60;
    labels.push(s===0?m+'m':m+':'+String(s).padStart(2,'0'));
    seriesList.forEach((samples,i)=>data[i].push(nearest(samples,t,key,20)));
  }}
  return{{labels,data}};
}}
function mkNet(canvasId,...series){{
  const rx=overlayAll(series,'rx');
  const rate=overlayAll(series,'rx_rate');
  const datasets=[];
  series.forEach((_,i)=>{{
    datasets.push({{label:SERIES_LABELS[i]+' RX (Mi)',data:rx.data[i],borderColor:COLOR_RX[i],
      backgroundColor:COLOR_RX[i]+'1a',tension:.25,pointRadius:1,fill:true,yAxisID:'y'}});
    datasets.push({{label:SERIES_LABELS[i]+' RX (Mi/s)',data:rate.data[i],borderColor:COLOR_RX[i],
      backgroundColor:'transparent',tension:.25,pointRadius:1,fill:false,borderDash:[4,3],yAxisID:'yRate'}});
  }});
  new Chart(document.getElementById(canvasId),{{type:'line',
    data:{{labels:rx.labels,datasets}},
    options:{{responsive:true,maintainAspectRatio:false,
      scales:{{y:{{beginAtZero:true,title:{{display:true,text:'Cumulative Mi'}}}},
              yRate:{{beginAtZero:true,position:'right',grid:{{drawOnChartArea:false}},
                title:{{display:true,text:'Mi/s'}}}},
              x:{{title:{{display:true,text:'Elapsed'}}}}}},
      plugins:{{legend:{{position:'bottom'}}}},interaction:{{mode:'index',intersect:false}}}}
  }});
}}

{chr(10).join(chart_calls)}
"""

    chart_js = load_chart_js()
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title}</title>
<script>
/* chart.js 4.4.1 UMD (vendored) */
{chart_js}
</script>
<style>
*,*::before,*::after{{box-sizing:border-box}}
body{{margin:0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  background:#f8f9fa;color:#1a1a1a;line-height:1.5}}
.wrap{{max-width:1100px;margin:0 auto;padding:24px}}
h1{{font-size:22px;margin:0 0 4px}}
h2{{font-size:16px;margin:32px 0 8px;border-bottom:1px solid #dee2e6;padding-bottom:4px}}
.meta{{color:#666;font-size:13px;margin-bottom:24px}}
.chart-box{{position:relative;height:280px;margin-bottom:8px}}
.chart-box.short{{height:240px}}
.caption{{color:#888;font-size:12px;margin-bottom:16px}}
table{{border-collapse:collapse;width:100%;margin-bottom:16px;font-size:14px}}
th,td{{text-align:left;padding:6px 12px;border-bottom:1px solid #dee2e6}}
th{{background:#f1f3f5;font-weight:600}}
hr{{border:none;border-top:1px solid #dee2e6;margin:24px 0}}
</style>
</head>
<body>
<div class="wrap">
<h1>{title}</h1>
<p class="meta">
  Memory = cgroup RSS (Mi) · CPU = metrics-server (millicores) · Net = cumulative RX (Mi) and per-sample rate (Mi/s)<br>
  {meta_line}
</p>
{lifetime_section}
{duration_section}
{"".join(mem_sections)}
<hr>

<h2>Peak resource usage</h2>
<table>
<tr><th>VM</th>{peak_head}</tr>
{"".join(peak_rows)}
</table>

<hr>

{"".join(net_sections)}
<h2>Network totals (last sample RX + peak Mi/s)</h2>
<table>
<tr><th>VM</th>{net_head}</tr>
{"".join(net_rows)}
</table>

</div>
<script>
{app_js}
</script>
</body>
</html>
"""


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] in ("-h", "--help"):
        print(__doc__.strip(), file=sys.stderr)
        return 2

    prefix = Path(sys.argv[1]).resolve()
    data: dict[str, dict[str, list[dict[str, Any]]]] = {}
    meta: dict[str, dict[str, str]] = {}
    converters: list[str] = []
    mem_dirs: list[Path] = []

    for conv in ("kc", "ref"):
        mem_dir = Path(f"{prefix}-{conv}-mem")
        if not mem_dir.is_dir():
            continue
        converters.append(conv)
        mem_dirs.append(mem_dir)

    if not converters:
        print(f"ERROR: no *-mem dirs found for prefix {prefix}", file=sys.stderr)
        return 1

    vm_labels = discover_vm_labels(mem_dirs)
    if not vm_labels:
        print(f"ERROR: no *-virt-v2v-memory.csv files under {prefix}-*-mem", file=sys.stderr)
        return 1

    for conv, mem_dir in zip(converters, mem_dirs):
        data[conv] = {
            vm: load_csv(mem_dir / f"{vm}{CSV_SUFFIX}") for vm in vm_labels
        }
        meta[conv] = parse_log_meta(Path(f"{prefix}-{conv}.log"), vm_labels)

    # Prefer compare title when both present
    if "ref" in converters and "kc" in converters:
        title = "Conversion pod — ref vs kc-v2v"
        mode = "compare"
    elif "kc" in converters:
        title = "Conversion pod — kc-v2v"
        mode = "kc"
    else:
        title = "Conversion pod — ref (virt-v2v)"
        mode = "ref"

    images = []
    for c in converters:
        img = meta.get(c, {}).get("image")
        if img:
            images.append(f"{c}={img}")
    vms: list[str] = []
    for vm in vm_labels:
        for c in converters:
            name = meta.get(c, {}).get(f"{vm}_vm")
            if name:
                vms.append(name)
                break

    lifetimes: list[str] = []
    for conv in converters:
        m = meta.get(conv, {})
        life = m.get("lifetime")
        started = m.get("started_at")
        ended = m.get("ended_at")
        if not life and not started:
            continue
        bit = f"{conv} lifetime={life or '—'}"
        if started and ended:
            bit += f" ({started} → {ended})"
        elif started:
            bit += f" (start={started})"
        lifetimes.append(bit)

    meta_line = (
        f"run {prefix.name} · mode={mode} · converters={','.join(converters)}"
        + (f" · VMs: {', '.join(vms)}" if vms else "")
        + (f" · {' · '.join(images)}" if images else "")
        + (f" · {' · '.join(lifetimes)}" if lifetimes else "")
    )

    out = Path(f"{prefix}.html")
    out.write_text(build_html(title, meta_line, converters, data, meta, vm_labels))
    print(f"Wrote {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
