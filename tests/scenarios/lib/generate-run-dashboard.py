#!/usr/bin/env python3
"""Generate a self-contained Chart.js dashboard for one MTV benchmark run.

Usage:
  generate-run-dashboard.py <artifact-prefix>

Looks for:
  <prefix>-kc-mem/{rhel,win}-virt-v2v-memory.csv
  <prefix>-ref-mem/{rhel,win}-virt-v2v-memory.csv
  <prefix>-{kc,ref}.log  (optional; peaks already come from CSVs)

Writes:
  <prefix>.html
"""

from __future__ import annotations

import csv
import json
import re
import sys
from pathlib import Path
from typing import Any


def load_csv(path: Path) -> list[dict[str, Any]]:
    if not path.is_file():
        return []
    rows: list[dict[str, Any]] = []
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
                rx = int(int(float(rx_raw)) / 1024 / 1024) if rx_raw != "" else None
            except ValueError:
                rx = None
            rows.append({"t": t, "mem": mem, "cpu": cpu, "rx": rx})
    return rows


def peak(samples: list[dict[str, Any]], key: str) -> int | None:
    vals = [s[key] for s in samples if s.get(key) is not None]
    return max(vals) if vals else None


def last(samples: list[dict[str, Any]], key: str) -> int | None:
    for s in reversed(samples):
        if s.get(key) is not None:
            return s[key]
    return None


def parse_log_meta(log_path: Path) -> dict[str, str]:
    meta: dict[str, str] = {}
    if not log_path.is_file():
        return meta
    text = log_path.read_text(errors="replace")
    m = re.search(r"virt_v2v_image_fqin:\s*(\S+)", text)
    if m:
        meta["image"] = m.group(1)
    m = re.search(r"^RHEL_VM=(\S+)", text, re.M)
    if m:
        meta["rhel_vm"] = m.group(1)
    m = re.search(r"^WIN_VM=(\S+)", text, re.M)
    if m:
        meta["win_vm"] = m.group(1)
    for label in ("rhel", "win"):
        m = re.search(
            rf"RESULT {label}: status=(\S+) total=(\S+)",
            text,
        )
        if m:
            meta[f"{label}_status"] = m.group(1)
            meta[f"{label}_wall"] = m.group(2)
    return meta


def fmt_peak(v: int | None, unit: str) -> str:
    return f"{v} {unit}" if v is not None else "—"


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
        for vm in ("rhel", "win"):
            key = f"{vm.upper()}_{conv.upper()}"
            consts.append(series_js(key, data.get(conv, {}).get(vm, [])))

    # Duration / peak / net table rows
    show_dur_diff = "ref" in order and "kc" in order
    dur_rows = []
    peak_rows = []
    net_rows = []
    for vm, vm_label in (("rhel", "RHEL"), ("win", "Windows")):
        dur_cells = [f"<td>{vm_label}</td>"]
        peak_cells = [f"<td>{vm_label}</td>"]
        net_cells = [f"<td>{vm_label}</td>"]
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
    net_head = "".join(f"<th>{labels.get(c, c)} RX</th>" for c in order)

    has_duration = any(
        meta.get(c, {}).get(f"{vm}_wall")
        for c in order
        for vm in ("rhel", "win")
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
    # Chart bootstrap JS — works for 1 or 2 converters
    series_args = {
        "rhel": ", ".join(f"{'rhel'.upper()}_{c.upper()}" for c in order),
        "win": ", ".join(f"{'win'.upper()}_{c.upper()}" for c in order),
    }
    legend_names = [labels.get(c, c) for c in order]
    color_mem = [colors[c]["mem"] for c in order]
    color_cpu = [colors[c]["cpu"] for c in order]
    color_rx = [colors[c]["rx"] for c in order]

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
function mkNet(canvasId,...series){{
  const rx=overlay(series,'rx');
  const datasets=series.map((_,i)=>({{
    label:SERIES_LABELS[i]+' RX (Mi)',data:rx.data[i],borderColor:COLOR_RX[i],
    backgroundColor:COLOR_RX[i]+'1a',tension:.3,pointRadius:1,fill:true
  }}));
  new Chart(document.getElementById(canvasId),{{type:'line',
    data:{{labels:rx.labels,datasets}},
    options:{{responsive:true,maintainAspectRatio:false,
      scales:{{y:{{beginAtZero:true,title:{{display:true,text:'Cumulative Mi'}}}},
              x:{{title:{{display:true,text:'Elapsed'}}}}}},
      plugins:{{legend:{{position:'bottom'}}}},interaction:{{mode:'index',intersect:false}}}}
  }});
}}

mkMemCpu('rhelMemCpu', {series_args["rhel"]});
mkMemCpu('winMemCpu', {series_args["win"]});
mkNet('rhelNet', {series_args["rhel"]});
mkNet('winNet', {series_args["win"]});
"""

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title}</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
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
  Memory = cgroup RSS (Mi) · CPU = metrics-server (millicores) · Net = cumulative /proc/net/dev (Mi)<br>
  {meta_line}
</p>
{duration_section}
<h2>RHEL conversion pod — memory &amp; CPU over time</h2>
<div class="chart-box"><canvas id="rhelMemCpu"></canvas></div>
<p class="caption">x-axis = elapsed seconds since monitor start</p>

<h2>Windows conversion pod — memory &amp; CPU over time</h2>
<div class="chart-box"><canvas id="winMemCpu"></canvas></div>

<hr>

<h2>Peak resource usage</h2>
<table>
<tr><th>VM</th>{peak_head}</tr>
{"".join(peak_rows)}
</table>

<hr>

<h2>RHEL conversion pod — network RX over time</h2>
<div class="chart-box short"><canvas id="rhelNet"></canvas></div>
<p class="caption">Cumulative bytes received from /proc/net/dev (all non-lo interfaces)</p>

<h2>Windows conversion pod — network RX over time</h2>
<div class="chart-box short"><canvas id="winNet"></canvas></div>

<h2>Network totals (last sample RX)</h2>
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

    for conv in ("kc", "ref"):
        mem_dir = Path(f"{prefix}-{conv}-mem")
        if not mem_dir.is_dir():
            continue
        converters.append(conv)
        data[conv] = {
            "rhel": load_csv(mem_dir / "rhel-virt-v2v-memory.csv"),
            "win": load_csv(mem_dir / "win-virt-v2v-memory.csv"),
        }
        meta[conv] = parse_log_meta(Path(f"{prefix}-{conv}.log"))

    if not converters:
        print(f"ERROR: no *-mem dirs found for prefix {prefix}", file=sys.stderr)
        return 1

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
    vms = []
    for c in converters:
        m = meta.get(c, {})
        if m.get("rhel_vm"):
            vms.append(m["rhel_vm"])
            break
    for c in converters:
        m = meta.get(c, {})
        if m.get("win_vm"):
            vms.append(m["win_vm"])
            break

    meta_line = (
        f"run {prefix.name} · mode={mode} · converters={','.join(converters)}"
        + (f" · VMs: {', '.join(vms)}" if vms else "")
        + (f" · {' · '.join(images)}" if images else "")
    )

    out = Path(f"{prefix}.html")
    out.write_text(build_html(title, meta_line, converters, data, meta))
    print(f"Wrote {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
