package webui

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ProxyGate — 智能代理网关</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&family=Noto+Sans+SC:wght@400;500;700&family=JetBrains+Mono:wght@500;600;700&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#f2ebd9;
  --bg-soft:#f7f1e6;
  --bg-elevated:#f8f3e8;
  --bg-card:rgba(253,251,247,0.92);
  --bg-card-solid:#fdfbf7;
  --bg-muted:#efe4d2;
  --bg-hover:#f4ebdc;
  --fg:#1c1917;
  --fg-dim:#5f574d;
  --fg-soft:#8a7f72;
  --border:#e8ddd4;
  --border-strong:#d8c8b8;
  --primary:#2d6a4f;
  --primary-strong:#1b4332;
  --primary-soft:#eef5f1;
  --green:#7cb08a;
  --yellow:#e0b15e;
  --orange:#de9963;
  --red:#d97868;
  --red-soft:#fdf2f1;
  --shadow-xs:0 2px 8px rgba(67,56,42,0.06);
  --shadow-sm:0 8px 24px rgba(67,56,42,0.08);
  --shadow-md:0 18px 40px rgba(67,56,42,0.1),0 4px 12px rgba(67,56,42,0.05);
  --shadow-lg:0 28px 60px rgba(67,56,42,0.12),0 8px 24px rgba(67,56,42,0.06);
  --radius-sm:14px;
  --radius-md:18px;
  --radius-lg:24px;
  --radius-xl:28px;
  --radius-pill:999px;
  --mono:"JetBrains Mono",ui-monospace,monospace;
  --sans:"Nunito","Noto Sans SC",system-ui,sans-serif;
}
html,body{min-height:100%}
body{
  background:radial-gradient(circle at top,#fbf6ec 0%,var(--bg) 54%,#e9dcc4 100%);
  color:var(--fg);
  font-family:var(--sans);
  font-size:14px;
  line-height:1.6;
  -webkit-font-smoothing:antialiased;
  position:relative;
}
body::before{
  content:'';
  position:fixed;
  inset:0;
  background:
    radial-gradient(circle at 86% 12%,rgba(224,177,94,0.18),transparent 22%),
    radial-gradient(circle at 14% 82%,rgba(45,106,79,0.12),transparent 24%);
  pointer-events:none;
}
body::after{
  content:'';
  position:fixed;
  inset:auto auto -180px -120px;
  width:420px;
  height:420px;
  border-radius:50%;
  background:radial-gradient(circle,rgba(45,106,79,0.14) 0%,rgba(45,106,79,0) 70%);
  pointer-events:none;
}
a{color:inherit}
button,select,input{font:inherit}
.layout{max-width:1760px;margin:0 auto;padding:24px 28px 32px;position:relative;z-index:1}
.content-grid{display:grid;grid-template-columns:minmax(0,1fr) 400px;gap:24px;align-items:start}
.main-content,.sidebar{min-width:0}
.sidebar{position:sticky;top:24px}
.proxy-section{display:flex;flex-direction:column;gap:24px}
.proxy-header{
  position:sticky;
  top:18px;
  z-index:100;
  display:flex;
  align-items:flex-start;
  justify-content:space-between;
  gap:20px;
  padding:24px 28px;
  background:rgba(253,251,247,0.82);
  border:1px solid rgba(232,221,212,0.9);
  border-radius:var(--radius-xl);
  box-shadow:var(--shadow-sm);
  backdrop-filter:blur(16px);
}
.proxy-logo-area{display:flex;align-items:flex-end;gap:16px;flex-wrap:wrap}
.proxy-logo-block{display:flex;flex-direction:column;gap:6px}
.proxy-tag{
  display:inline-flex;
  align-items:center;
  width:max-content;
  padding:6px 12px;
  border-radius:var(--radius-pill);
  background:var(--primary-soft);
  color:var(--primary);
  font-size:11px;
  font-weight:800;
  letter-spacing:0.08em;
  text-transform:uppercase;
}
.proxy-logo{
  font-size:clamp(30px,4.2vw,44px);
  line-height:1;
  font-weight:800;
  letter-spacing:-0.04em;
  color:var(--fg);
}
.proxy-subtitle{font-size:13px;color:var(--fg-dim)}
.user-badge{
  display:inline-flex;
  align-items:center;
  justify-content:center;
  min-height:38px;
  padding:0 14px;
  border-radius:var(--radius-pill);
  background:var(--bg-muted);
  color:var(--fg-dim);
  font-family:var(--mono);
  font-size:11px;
  font-weight:700;
  letter-spacing:0.04em;
  text-transform:uppercase;
}
.header-actions{display:flex;gap:10px;align-items:center;justify-content:flex-end;flex-wrap:wrap}
.proxy-content{min-width:0}
.tab,.filter-select{
  min-height:42px;
  padding:0 16px;
  border-radius:var(--radius-pill);
  border:1px solid var(--border);
  background:var(--bg-card-solid);
  color:var(--fg-dim);
  display:inline-flex;
  align-items:center;
  justify-content:center;
  text-decoration:none;
  font-size:12px;
  font-weight:700;
  letter-spacing:0.02em;
  transition:all 0.2s ease;
  box-shadow:var(--shadow-xs);
}
.tab:hover,.filter-select:hover{
  transform:translateY(-1px);
  border-color:var(--border-strong);
  background:var(--bg-hover);
  color:var(--fg);
  box-shadow:var(--shadow-sm);
}
.tab:active{transform:translateY(0)}
.tab-accent{
  color:var(--primary);
  border-color:rgba(45,106,79,0.16);
  background:var(--primary-soft);
}
.icon-tab{width:42px;padding:0}
.filter-select{
  appearance:none;
  padding-right:40px;
  background-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%232d6a4f' d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat:no-repeat;
  background-position:right 14px center;
}
.filter-select:focus{
  outline:none;
  border-color:rgba(45,106,79,0.42);
  box-shadow:0 0 0 4px rgba(45,106,79,0.12);
}
.filter-select option{background:var(--bg-card-solid);color:var(--fg)}
#proxy-table-wrap{
  background:var(--bg-card-solid);
  border:1px solid var(--border);
  border-radius:var(--radius-xl);
  box-shadow:var(--shadow-md);
  overflow:auto;
}
table{
  width:100%;
  min-width:920px;
  border-collapse:collapse;
  background:transparent;
}
thead{background:transparent}
th{
  position:sticky;
  top:0;
  z-index:10;
  padding:16px 18px;
  text-align:left;
  font-size:11px;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--fg-soft);
  font-weight:800;
  background:rgba(248,243,232,0.96);
  backdrop-filter:blur(10px);
  box-shadow:inset 0 -1px 0 var(--border);
}
td{
  padding:16px 18px;
  border-top:1px solid rgba(232,221,212,0.78);
  color:var(--fg-dim);
  vertical-align:middle;
}
tr:hover{background:rgba(45,106,79,0.03)}
.cell-mono{font-family:var(--mono);font-size:12px}
.cell-grade{font-weight:800;font-size:16px}
.cell-clickable{cursor:pointer;transition:background 0.2s ease,color 0.2s ease}
.cell-clickable:hover{background:var(--primary-soft)!important;color:var(--primary-strong)!important}
.cell-clickable:active{background:rgba(45,106,79,0.18)!important}
.grade-s{color:var(--primary-strong)}
.grade-a{color:#9a6a1d}
.grade-b{color:#b96d35}
.grade-c{color:var(--red)}
.badge{
  display:inline-flex;
  align-items:center;
  padding:5px 10px;
  border-radius:var(--radius-pill);
  font-size:10px;
  font-weight:800;
  text-transform:uppercase;
  letter-spacing:0.08em;
  border:1px solid transparent;
  font-family:var(--mono);
}
.badge-http{
  border-color:rgba(139,115,85,0.18);
  color:var(--fg-dim);
  background:rgba(139,115,85,0.08);
}
.badge-socks5{
  background:var(--primary-soft);
  color:var(--primary-strong);
  border-color:rgba(45,106,79,0.16);
}
.source-badge{
  display:inline-flex;
  align-items:center;
  padding:4px 8px;
  margin-left:6px;
  border-radius:var(--radius-pill);
  background:rgba(224,177,94,0.22);
  color:#8a6215;
  font-size:10px;
  font-weight:800;
  letter-spacing:0.04em;
}
.row-custom td:first-child{box-shadow:inset 3px 0 0 var(--yellow)}
.control-panel,.quality-bar,.sidebar .section{
  background:var(--bg-card-solid);
  border:1px solid var(--border);
  border-radius:var(--radius-xl);
  padding:20px;
  box-shadow:var(--shadow-sm);
}
.sidebar>*:not(:last-child){margin-bottom:18px}
.control-header,.section-header{
  display:flex;
  align-items:center;
  justify-content:space-between;
  margin-bottom:16px;
  padding-bottom:12px;
  border-bottom:1px solid var(--border);
}
.control-title,.section-title,.quality-bar-title{
  font-size:12px;
  font-weight:800;
  letter-spacing:0.08em;
  text-transform:uppercase;
  color:var(--fg);
}
.control-ops{display:flex;gap:10px;flex-wrap:wrap}
.ctrl-btn-primary,.ctrl-btn-secondary,.modal-actions .btn{
  flex:1;
  min-height:44px;
  padding:0 16px;
  border-radius:var(--radius-pill);
  border:1px solid transparent;
  cursor:pointer;
  transition:all 0.2s ease;
  font-size:12px;
  font-weight:800;
  letter-spacing:0.03em;
}
.ctrl-btn-primary,.modal-actions .btn{
  background:linear-gradient(135deg,var(--primary),var(--primary-strong));
  color:#fff;
  box-shadow:var(--shadow-xs);
}
.ctrl-btn-primary:hover,.modal-actions .btn:hover{
  transform:translateY(-1px);
  box-shadow:var(--shadow-sm);
}
.ctrl-btn-secondary,.modal-actions .btn-secondary{
  background:var(--bg-card-solid);
  color:var(--fg-dim);
  border-color:var(--border);
}
.ctrl-btn-secondary:hover,.modal-actions .btn-secondary:hover{
  background:var(--bg-hover);
  border-color:var(--border-strong);
  color:var(--fg);
}
.panel-label{
  margin:4px 0 10px;
  font-size:11px;
  font-weight:800;
  letter-spacing:0.08em;
  text-transform:uppercase;
  color:var(--primary);
}
.panel-label.accent{color:#9a6a1d}
.health-grid{
  display:grid;
  grid-template-columns:repeat(2,minmax(0,1fr));
  gap:14px;
  background:transparent;
  border:none;
  box-shadow:none;
}
.health-grid.health-grid-wide{grid-template-columns:repeat(3,minmax(0,1fr))}
.health-card{
  position:relative;
  padding:18px 18px 16px;
  background:var(--bg-card-solid);
  border:1px solid var(--border);
  border-radius:var(--radius-lg);
  box-shadow:var(--shadow-xs);
  overflow:hidden;
}
.health-card::before{
  content:'';
  position:absolute;
  inset:0 auto 0 0;
  width:4px;
  background:rgba(45,106,79,0.18);
}
.health-label{
  font-size:11px;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--fg-soft);
  margin-bottom:8px;
  font-weight:800;
}
.health-value{
  font-size:28px;
  font-weight:800;
  line-height:1;
  letter-spacing:-0.04em;
  color:var(--fg);
}
.health-value.health-state{font-size:20px;text-transform:uppercase}
.health-meta{
  font-size:11px;
  color:var(--fg-soft);
  margin-top:8px;
  font-family:var(--mono);
}
.health-status{
  position:absolute;
  top:18px;
  right:18px;
  width:10px;
  height:10px;
  border-radius:50%;
}
.health-status.healthy{background:var(--green);box-shadow:0 0 0 4px rgba(124,176,138,0.18)}
.health-status.warning{background:var(--yellow);box-shadow:0 0 0 4px rgba(224,177,94,0.18)}
.health-status.critical{background:var(--orange);box-shadow:0 0 0 4px rgba(222,153,99,0.18)}
.health-status.emergency{background:var(--red);box-shadow:0 0 0 4px rgba(217,120,104,0.18);animation:pulse 1s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:0.6}}
.quality-bar-title{margin-bottom:14px}
.quality-visual{
  display:flex;
  height:22px;
  border-radius:var(--radius-pill);
  overflow:hidden;
  background:var(--bg-soft);
  border:1px solid var(--border);
}
.quality-segment{
  display:flex;
  align-items:center;
  justify-content:center;
  font-size:10px;
  font-weight:800;
  font-family:var(--mono);
  color:#fff;
  transition:width 0.3s ease;
}
.quality-s{background:linear-gradient(135deg,var(--primary),var(--primary-strong))}
.quality-a{background:#a6752d}
.quality-b{background:#c67b43}
.quality-c{background:var(--red)}
.quality-legend{
  display:grid;
  grid-template-columns:repeat(2,minmax(0,1fr));
  gap:10px;
  margin-top:14px;
}
.quality-legend-item{font-size:11px;color:var(--fg-dim)}
.quality-legend-dot{
  display:inline-block;
  width:8px;
  height:8px;
  margin-right:6px;
  border-radius:50%;
}
.btn-danger,.btn-action{
  display:inline-flex;
  align-items:center;
  justify-content:center;
  min-height:32px;
  padding:0 12px;
  border-radius:var(--radius-pill);
  font-size:10px;
  font-weight:800;
  letter-spacing:0.04em;
  cursor:pointer;
  transition:all 0.2s ease;
  background:var(--bg-card-solid);
}
.btn-action{
  border:1px solid var(--border);
  color:var(--fg-dim);
  margin-left:8px;
}
.btn-action:hover{background:var(--bg-hover);border-color:var(--border-strong);color:var(--fg)}
.btn-danger{
  border:1px solid rgba(217,120,104,0.28);
  color:var(--red);
  background:var(--red-soft);
}
.btn-danger:hover{background:var(--red);border-color:var(--red);color:#fff}
.modal-overlay{
  display:none;
  position:fixed;
  inset:0;
  padding:24px;
  background:rgba(28,25,23,0.18);
  backdrop-filter:blur(10px);
  z-index:100;
  align-items:center;
  justify-content:center;
}
.modal-overlay.show{display:flex}
.modal{
  background:var(--bg-card-solid);
  border:1px solid var(--border);
  border-radius:32px;
  padding:32px;
  width:min(700px,100%);
  box-shadow:var(--shadow-lg);
  max-height:90vh;
  overflow-y:auto;
}
.modal-title{
  font-size:26px;
  font-weight:800;
  letter-spacing:-0.03em;
  color:var(--fg);
  margin-bottom:24px;
}
.form-section{margin-bottom:24px}
.form-section-title{
  font-size:11px;
  text-transform:uppercase;
  letter-spacing:0.08em;
  color:var(--primary);
  margin-bottom:14px;
  font-weight:800;
}
.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}
.form-group{display:flex;flex-direction:column;gap:8px}
.form-group label{font-size:12px;color:var(--fg);font-weight:700}
.form-group input,.form-select{
  width:100%;
  min-height:44px;
  padding:0 14px;
  border:1px solid var(--border);
  border-radius:var(--radius-md);
  background:var(--bg-soft);
  color:var(--fg);
  outline:none;
  transition:all 0.2s ease;
}
.form-group input:focus,.form-select:focus{
  border-color:rgba(45,106,79,0.42);
  background:var(--bg-card-solid);
  box-shadow:0 0 0 4px rgba(45,106,79,0.12);
}
.form-help{font-size:11px;color:var(--fg-soft)}
.modal-note{color:var(--fg-dim);font-size:13px;margin-bottom:18px;line-height:1.7}
.modal-note-sub{display:block;color:var(--fg-soft);font-size:12px;margin-top:6px}
.modal-actions{
  display:flex;
  gap:12px;
  margin-top:28px;
  padding-top:22px;
  border-top:1px solid var(--border);
}
.upload-dropzone{
  border:1px dashed var(--border-strong);
  border-radius:var(--radius-lg);
  padding:20px;
  text-align:center;
  cursor:pointer;
  transition:all 0.2s ease;
  background:rgba(255,255,255,0.46);
}
.upload-dropzone:hover{background:var(--primary-soft);border-color:rgba(45,106,79,0.35)}
.upload-label{color:var(--fg-dim);font-size:12px}
.file-selected{color:var(--primary-strong);font-weight:800}
.file-selected-meta{font-size:11px;color:var(--fg-soft)}
#sub-list{
  display:flex;
  flex-direction:column;
  gap:10px;
  max-height:240px;
  overflow-y:auto;
  padding-right:4px;
}
.sub-status{margin-top:10px;font-size:11px;color:var(--fg-soft)}
.subscription-empty{
  color:var(--fg-soft);
  text-align:center;
  padding:12px;
  border-radius:var(--radius-lg);
  background:var(--bg-soft);
}
.subscription-item{
  display:flex;
  align-items:flex-start;
  justify-content:space-between;
  gap:12px;
  padding:12px 14px;
  border:1px solid var(--border);
  border-radius:var(--radius-lg);
  background:var(--bg-soft);
}
.subscription-item-main{flex:1;min-width:0}
.subscription-name-row{
  display:flex;
  align-items:center;
  flex-wrap:wrap;
  gap:8px;
  margin-bottom:4px;
}
.subscription-status{font-size:12px;font-weight:800}
.subscription-status.active{color:var(--primary)}
.subscription-status.inactive{color:var(--fg-soft)}
.subscription-name{font-weight:800;color:var(--fg)}
.subscription-stats{font-size:11px;color:var(--fg-soft)}
.subscription-badge{
  display:inline-flex;
  align-items:center;
  padding:4px 8px;
  border-radius:var(--radius-pill);
  font-size:10px;
  font-weight:800;
}
.subscription-badge-warm{background:rgba(222,153,99,0.18);color:#9c5c24}
.subscription-actions{display:flex;gap:8px;flex-shrink:0}
.icon-btn{
  width:30px;
  height:30px;
  border-radius:50%;
  border:1px solid var(--border);
  background:var(--bg-card-solid);
  color:var(--fg-dim);
  display:inline-flex;
  align-items:center;
  justify-content:center;
  cursor:pointer;
  transition:all 0.2s ease;
}
.icon-btn:hover{background:var(--bg-hover);border-color:var(--border-strong);color:var(--fg)}
.icon-btn-danger{
  border-color:rgba(217,120,104,0.28);
  color:var(--red);
  background:var(--red-soft);
}
.icon-btn-danger:hover{background:var(--red);border-color:var(--red);color:#fff}
.log-box{
  padding:14px 16px;
  background:var(--bg-soft);
  border:1px solid var(--border);
  border-radius:var(--radius-lg);
  font-family:var(--mono);
  font-size:11px;
  color:var(--fg-dim);
  height:360px;
  overflow-y:auto;
  line-height:1.8;
}
.log-box::-webkit-scrollbar,#sub-list::-webkit-scrollbar,.modal::-webkit-scrollbar{width:6px}
.log-box::-webkit-scrollbar-thumb,#sub-list::-webkit-scrollbar-thumb,.modal::-webkit-scrollbar-thumb{
  background:rgba(138,127,114,0.36);
  border-radius:999px;
}
.log-box::-webkit-scrollbar-track,#sub-list::-webkit-scrollbar-track,.modal::-webkit-scrollbar-track{background:transparent}
.log-line{padding:3px 0;opacity:0.88}
.log-line.error{color:var(--red);font-weight:700}
.log-line.success{color:var(--primary)}
.log-meta{
  font-size:11px;
  color:var(--fg-soft);
  font-family:var(--mono);
  margin-top:10px;
  text-align:center;
}
.empty{
  padding:56px 24px;
  text-align:center;
  color:var(--fg-soft);
  font-size:13px;
}
.admin-only{display:none}
.toast{
  position:fixed;
  bottom:32px;
  left:50%;
  transform:translateX(-50%) translateY(100px);
  background:linear-gradient(135deg,var(--primary),var(--primary-strong));
  color:#fff;
  padding:12px 22px;
  border-radius:var(--radius-pill);
  font-size:12px;
  font-weight:800;
  font-family:var(--sans);
  opacity:0;
  transition:all 0.3s ease;
  z-index:1000;
  pointer-events:none;
  box-shadow:var(--shadow-md);
}
.toast.show{transform:translateX(-50%) translateY(0);opacity:1}
@media (max-width: 1280px) {
  .content-grid{grid-template-columns:1fr}
  .sidebar{position:static}
  .health-grid.health-grid-wide{grid-template-columns:repeat(3,minmax(0,1fr))}
}
@media (max-width: 900px) {
  .layout{padding:18px 16px 24px}
  .proxy-header{top:12px;padding:20px;flex-direction:column}
  .header-actions{width:100%;justify-content:flex-start}
  .form-grid{grid-template-columns:1fr}
  .modal{padding:24px}
  .modal-actions{flex-direction:column}
}
@media (max-width: 640px) {
  .health-grid,.health-grid.health-grid-wide{grid-template-columns:1fr}
  .control-ops{flex-direction:column}
  .tab,.filter-select{flex:1 1 calc(50% - 10px)}
  .subscription-item{flex-direction:column}
  .subscription-actions{width:100%;justify-content:flex-end}
}
</style>
</head>
<body>
<div class="layout">
  <div class="content-grid">
    <div class="main-content">
      <div class="proxy-section">
        <div class="proxy-header">
          <div class="proxy-logo-area">
            <div class="proxy-logo-block">
              <div class="proxy-tag">Unified Proxy Gateway</div>
              <div class="proxy-logo">ProxyGate</div>
            </div>
            <span id="user-mode" class="user-badge">Guest</span>
          </div>
          <div class="header-actions">
            <select class="filter-select" id="protocol-filter" onchange="setProtocolFilter(this.value)">
              <option value="" id="protocol-filter-label">协议</option>
              <option value="http">HTTP</option>
              <option value="socks5">SOCKS5</option>
            </select>
            <select class="filter-select" id="country-filter" onchange="setCountryFilter(this.value)">
              <option value="" id="country-filter-label">出口国家</option>
            </select>
            <button class="tab" onclick="toggleLang()" id="lang-btn">EN</button>
            <a href="https://github.com/ruruamour/ProxyGate" target="_blank" class="tab icon-tab" title="GitHub">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" style="vertical-align: middle;">
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
              </svg>
            </a>
            <button class="tab tab-accent guest-only" onclick="openContributeModal()" data-i18n="contribute.nav">贡献订阅</button>
            <a href="/login" class="tab" id="login-link" style="display: none;" data-i18n="nav.login">登录</a>
            <a href="/logout" class="tab admin-only" data-i18n="nav.logout">退出</a>
            <button class="tab icon-tab admin-only" onclick="openSettings()" title="" data-i18n-title="contribute.settings">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
            </button>
          </div>
        </div>
        <div class="proxy-content">
          <div id="proxy-table-wrap"><div class="empty" data-i18n="proxy.loading">加载中...</div></div>
        </div>
      </div>
    </div>

    <aside class="sidebar">
      <div class="control-panel admin-only">
        <div class="control-header">
          <div class="control-title" data-i18n="control.title">控制中心</div>
        </div>
        <div class="control-ops">
          <button class="ctrl-btn-primary" onclick="triggerFetch()" data-i18n="actions.fetch">抓取代理</button>
          <button class="ctrl-btn-secondary" onclick="refreshLatency()" data-i18n="actions.refresh">刷新延迟</button>
          <!-- 配置按钮已移到顶部导航 -->
        </div>
      </div>

      <!-- 订阅管理面板 -->
      <div class="control-panel admin-only" style="margin-bottom:20px">
        <div class="control-header">
          <div class="control-title" data-i18n="sub.title">订阅管理</div>
        </div>
        <div id="sub-list"></div>
        <div class="control-ops">
          <button class="ctrl-btn-primary" onclick="openSubModal()" data-i18n="sub.add">添加订阅</button>
          <button class="ctrl-btn-secondary" onclick="refreshAllSubs()" data-i18n="sub.refresh_all">刷新所有订阅</button>
        </div>
        <div id="sub-status" class="sub-status"></div>
      </div>

      <!-- 免费代理池 -->
      <div class="panel-label" data-i18n="health.free_pool">免费池</div>
      <div class="health-grid">
        <div class="health-card">
          <div class="health-label" data-i18n="health.status">池子状态</div>
          <div class="health-value health-state" id="pool-state">—</div>
          <div class="health-status" id="pool-status-dot"></div>
        </div>
        <div class="health-card">
          <div class="health-label" data-i18n="health.total">免费代理</div>
          <div class="health-value" id="stat-total">0</div>
          <div class="health-meta"><span id="stat-capacity">0</span> <span data-i18n="health.capacity">容量</span></div>
        </div>
        <div class="health-card">
          <div class="health-label">HTTP</div>
          <div class="health-value" id="stat-http">0</div>
          <div class="health-meta"><span id="http-slots">0</span> <span data-i18n="health.slots">槽位</span> · <span id="http-avg">—</span>ms <span data-i18n="health.avg">平均</span></div>
        </div>
        <div class="health-card">
          <div class="health-label">SOCKS5</div>
          <div class="health-value" id="stat-socks5">0</div>
          <div class="health-meta"><span id="socks5-slots">0</span> <span data-i18n="health.slots">槽位</span> · <span id="socks5-avg">—</span>ms <span data-i18n="health.avg">平均</span></div>
        </div>
      </div>

      <!-- 订阅代理池 -->
      <div class="panel-label accent" data-i18n="health.sub_pool">订阅池</div>
      <div class="health-grid health-grid-wide">
        <div class="health-card">
          <div class="health-label" data-i18n="health.sub_sources">订阅源</div>
          <div class="health-value" id="stat-sub-count">0</div>
          <div class="health-meta" id="stat-sub-meta">—</div>
        </div>
        <div class="health-card">
          <div class="health-label" data-i18n="health.available">可用</div>
          <div class="health-value" id="stat-custom">0</div>
          <div class="health-meta" id="custom-meta">—</div>
        </div>
        <div class="health-card">
          <div class="health-label" data-i18n="health.disabled">禁用/待恢复</div>
          <div class="health-value" id="stat-custom-disabled">0</div>
          <div class="health-meta" id="custom-disabled-meta" data-i18n="health.awaiting_probe">探测唤醒中</div>
        </div>
      </div>

      <div class="quality-bar">
        <div class="quality-bar-title" data-i18n="quality.title">质量分布</div>
        <div class="quality-visual" id="quality-visual">
          <div class="quality-segment quality-s" style="width:0%"></div>
          <div class="quality-segment quality-a" style="width:0%"></div>
          <div class="quality-segment quality-b" style="width:0%"></div>
          <div class="quality-segment quality-c" style="width:0%"></div>
        </div>
        <div class="quality-legend">
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:var(--primary)"></span><span data-i18n="quality.grade_s">S级</span> (<span id="grade-s-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#a6752d"></span><span data-i18n="quality.grade_a">A级</span> (<span id="grade-a-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#c67b43"></span><span data-i18n="quality.grade_b">B级</span> (<span id="grade-b-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:var(--red)"></span><span data-i18n="quality.grade_c">C级</span> (<span id="grade-c-count">0</span>)</div>
        </div>
      </div>

      <div class="section">
        <div class="section-header">
          <h2 class="section-title" data-i18n="log.title">系统日志</h2>
        </div>
        <div class="log-box" id="logs-box"><span data-i18n="log.loading">加载中...</span></div>
        <div class="log-meta">
          <span data-i18n="log.auto_refresh_label">自动刷新</span>: <span id="log-countdown">5</span>s
        </div>
      </div>
    </aside>
  </div>
</div>

<div class="modal-overlay" id="settings-modal" onclick="if(event.target===this) closeSettings()">
  <div class="modal">
    <div class="modal-title" data-i18n="config.system_title">系统设置</div>

    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_proxy_mode">代理使用模式</div>
      <div class="form-grid">
        <div class="form-group" style="grid-column:1/-1">
          <label data-i18n="config.proxy_strategy">出站代理选择策略</label>
          <select id="cfg-custom-mode" class="form-select">
            <option value="mixed_custom_priority" data-i18n="config.mode_mixed_custom">混合 · 订阅优先</option>
            <option value="mixed_free_priority" data-i18n="config.mode_mixed_free">混合 · 免费优先</option>
            <option value="mixed" data-i18n="config.mode_mixed">混合 · 平等</option>
            <option value="custom_only" data-i18n="config.mode_custom_only">仅订阅代理</option>
            <option value="free_only" data-i18n="config.mode_free_only">仅免费代理</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 免费池设置 -->
    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_free_pool">免费代理池</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.pool_capacity">池子容量</label>
          <input type="number" id="cfg-pool-size" min="10" max="500">
          <div class="form-help" data-i18n="config.pool_capacity_help">免费代理总槽位</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.http_ratio_label">HTTP 占比</label>
          <input type="number" id="cfg-http-ratio" min="0" max="1" step="0.05">
          <div class="form-help" data-i18n="config.http_ratio_help">0.3 = 30% HTTP</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.min_per_protocol">每协议最小数</label>
          <input type="number" id="cfg-min-per-protocol" min="1" max="50">
        </div>
        <div class="form-group">
          <label data-i18n="config.latency_standard">标准延迟 (ms)</label>
          <input type="number" id="cfg-max-latency" min="500" max="5000" step="100">
        </div>
        <div class="form-group">
          <label data-i18n="config.latency_healthy">健康延迟 (ms)</label>
          <input type="number" id="cfg-max-latency-healthy" min="500" max="3000" step="100">
        </div>
        <div class="form-group">
          <label data-i18n="config.latency_emergency">紧急延迟 (ms)</label>
          <input type="number" id="cfg-max-latency-emergency" min="1000" max="5000" step="100">
        </div>
        <div class="form-group">
          <label data-i18n="config.optimize_interval">优化间隔 (分钟)</label>
          <input type="number" id="cfg-optimize-interval" min="10" max="120" step="10">
        </div>
        <div class="form-group">
          <label data-i18n="config.replace_threshold">替换阈值</label>
          <input type="number" id="cfg-replace-threshold" min="0.5" max="0.9" step="0.05">
          <div class="form-help" data-i18n="config.replace_threshold_help">新代理需快30%</div>
        </div>
      </div>
    </div>

    <!-- 订阅池设置 -->
    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_sub_pool">订阅代理池</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.probe_interval">探测间隔 (分钟)</label>
          <input type="number" id="cfg-custom-probe" min="5" max="120" step="5">
          <div class="form-help" data-i18n="config.probe_interval_help">禁用代理的唤醒探测间隔</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.refresh_interval">默认刷新间隔 (分钟)</label>
          <input type="number" id="cfg-custom-refresh" min="10" max="1440" step="10">
          <div class="form-help" data-i18n="config.refresh_interval_help">新订阅的默认刷新周期</div>
        </div>
      </div>
    </div>

    <!-- 验证与检查 -->
    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_validation">验证与健康检查</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.validate_concurrency">验证并发数</label>
          <input type="number" id="cfg-concurrency" min="50" max="500" step="50">
        </div>
        <div class="form-group">
          <label data-i18n="config.validate_timeout">验证超时 (秒)</label>
          <input type="number" id="cfg-timeout" min="3" max="15">
        </div>
        <div class="form-group">
          <label data-i18n="config.health_interval">检查间隔 (分钟)</label>
          <input type="number" id="cfg-health-interval" min="1" max="60">
        </div>
        <div class="form-group">
          <label data-i18n="config.health_batch">每批数量</label>
          <input type="number" id="cfg-health-batch" min="10" max="100" step="10">
        </div>
      </div>
    </div>

    <!-- 地理过滤 -->
    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_geo_filter">地理过滤</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.allowed_countries">允许国家（白名单）</label>
          <input type="text" id="cfg-allowed-countries" placeholder="US,JP,KR,SG">
          <div class="form-help" data-i18n="config.allowed_countries_help">非空时仅允许这些国家，忽略黑名单</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.blocked_countries">屏蔽国家（黑名单）</label>
          <input type="text" id="cfg-blocked-countries" placeholder="CN,RU,KP">
          <div class="form-help" data-i18n="config.blocked_countries_help">白名单为空时生效</div>
        </div>
      </div>
    </div>

    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeSettings()" data-i18n="config.cancel">取消</button>
      <button class="btn" onclick="saveConfig()" data-i18n="config.save">保存配置</button>
    </div>
  </div>
</div>

<!-- 添加订阅弹窗 -->
<div class="modal-overlay" id="sub-modal" onclick="if(event.target===this) closeSubModal()" style="display:none">
  <div class="modal" style="max-width:500px">
    <div class="modal-title" data-i18n="sub.add_title">添加订阅</div>
    <div class="form-section">
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="sub.name">名称</label>
          <input type="text" id="sub-name" placeholder="">
        </div>
        <div class="form-group" style="grid-column:1/-1">
          <label data-i18n="sub.import_mode">导入方式</label>
          <div style="display:flex;gap:8px;margin-bottom:8px">
            <button id="tab-url" class="ctrl-btn-primary" onclick="switchSubTab('url')" style="flex:1" data-i18n="sub.tab_url">订阅 URL</button>
            <button id="tab-file" class="ctrl-btn-secondary" onclick="switchSubTab('file')" style="flex:1" data-i18n="sub.tab_file">上传文件</button>
          </div>
        </div>
        <div class="form-group" id="sub-url-group" style="grid-column:1/-1">
          <label data-i18n="sub.url_label">订阅 URL</label>
          <input type="text" id="sub-url" placeholder="https://example.com/sub?token=xxx">
          <div class="form-help" data-i18n="sub.url_help">自动识别格式</div>
        </div>
        <div class="form-group" id="sub-file-group" style="grid-column:1/-1;display:none">
          <label data-i18n="sub.file_label">配置文件</label>
          <div class="upload-dropzone"
               onclick="document.getElementById('sub-file-input').click()"
               ondragover="event.preventDefault();this.style.borderColor='var(--primary)'"
               ondragleave="this.style.borderColor='var(--border)'"
               ondrop="event.preventDefault();this.style.borderColor='var(--border)';handleFileDrop(event)">
            <div id="sub-file-label" class="upload-label" data-i18n="sub.file_drop">点击选择或拖拽文件到此处</div>
          </div>
          <input type="file" id="sub-file-input" accept=".yaml,.yml,.txt,.conf,.json" style="display:none" onchange="handleFileSelect(this)">
        </div>
        <div class="form-group">
          <label data-i18n="sub.refresh_min">刷新间隔 (分钟)</label>
          <input type="number" id="sub-refresh" value="60" min="10" max="1440" step="10">
          <div class="form-help" data-i18n="sub.refresh_min_help">仅 URL 模式有效</div>
        </div>
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeSubModal()" data-i18n="sub.cancel">取消</button>
      <button class="btn" onclick="addSubscription()" data-i18n="sub.submit">添加</button>
    </div>
  </div>
</div>

<!-- 访客贡献订阅弹窗 -->
<div class="modal-overlay" id="contribute-modal" onclick="if(event.target===this) closeContributeModal()" style="display:none">
  <div class="modal" style="max-width:460px">
    <div class="modal-title" data-i18n="contribute.title">贡献订阅</div>
    <div class="modal-note">
      <span data-i18n="contribute.desc">分享你的代理订阅，帮助丰富代理池。</span><br>
      <span class="modal-note-sub" data-i18n="contribute.privacy">你的订阅仅用于此代理池，不会被用于其他渠道。连续探测无可用节点将自动移除。</span>
    </div>
    <div class="form-section">
      <div class="form-grid">
        <div class="form-group" style="grid-column:1/-1">
          <label data-i18n="sub.name">名称</label>
          <input type="text" id="contribute-name" placeholder="">
        </div>
        <div class="form-group" style="grid-column:1/-1">
          <label data-i18n="sub.import_mode">导入方式</label>
          <div style="display:flex;gap:8px;margin-bottom:8px">
            <button id="ctab-url" class="ctrl-btn-primary" onclick="switchContributeTab('url')" style="flex:1" data-i18n="sub.tab_url">订阅 URL</button>
            <button id="ctab-file" class="ctrl-btn-secondary" onclick="switchContributeTab('file')" style="flex:1" data-i18n="sub.tab_file">上传文件</button>
          </div>
        </div>
        <div class="form-group" id="contribute-url-group" style="grid-column:1/-1">
          <label data-i18n="sub.url_label">订阅 URL</label>
          <input type="text" id="contribute-url" placeholder="https://example.com/sub?token=xxx">
          <div class="form-help" data-i18n="sub.url_help">自动识别格式</div>
        </div>
        <div class="form-group" id="contribute-file-group" style="grid-column:1/-1;display:none">
          <label data-i18n="sub.file_label">配置文件</label>
          <div class="upload-dropzone"
               onclick="document.getElementById('contribute-file-input').click()"
               ondragover="event.preventDefault();this.style.borderColor='var(--primary)'"
               ondragleave="this.style.borderColor='var(--border)'"
               ondrop="event.preventDefault();this.style.borderColor='var(--border)';handleContributeFileDrop(event)">
            <div id="contribute-file-label" class="upload-label" data-i18n="sub.file_drop">点击选择或拖拽文件到此处</div>
          </div>
          <input type="file" id="contribute-file-input" accept=".yaml,.yml,.txt,.conf,.json" style="display:none" onchange="handleContributeFileSelect(this)">
        </div>
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeContributeModal()" data-i18n="sub.cancel">取消</button>
      <button class="btn" id="contribute-submit-btn" onclick="submitContribution()" data-i18n="contribute.submit">提交</button>
    </div>
  </div>
</div>

<script>
// 国际化翻译
const i18n = {
  zh: {
    'control.title': '控制中心',
    'nav.config': '配置',
    'nav.login': '登录',
    'nav.logout': '退出',
    'health.status': '池子状态',
    'health.total': '总代理数',
    'health.capacity': '容量',
    'health.slots': '槽位',
    'health.avg': '平均',
    'health.state.healthy': '健康',
    'health.state.warning': '警告',
    'health.state.critical': '危急',
    'health.state.emergency': '紧急',
    'quality.title': '质量分布',
    'quality.grade_s': 'S级',
    'quality.grade_a': 'A级',
    'quality.grade_b': 'B级',
    'quality.grade_c': 'C级',
    'actions.fetch': '抓取代理',
    'actions.refresh': '刷新延迟',
    'actions.config': '配置池子',
    'proxy.title': '代理列表',
    'proxy.tab_all': '全部',
    'proxy.filter_protocol': '协议',
    'proxy.filter_country': '出口国家',
    'proxy.loading': '加载中...',
    'proxy.empty': '暂无代理',
    'proxy.th_grade': '等级',
    'proxy.th_protocol': '协议',
    'proxy.th_address': '地址',
    'proxy.th_exit_ip': '出口IP',
    'proxy.th_location': '位置',
    'proxy.th_latency': '延迟',
    'proxy.th_usage': '使用统计',
    'proxy.th_action': '操作',
    'proxy.btn_delete': '删除',
    'proxy.btn_refresh': '刷新',
    'proxy.copy_success': '已复制',
    'proxy.refresh_started': '刷新已启动',
    'log.title': '系统日志',
    'log.auto_refresh_label': '自动刷新',
    'log.loading': '加载中...',
    'log.empty': '暂无日志',
    'config.title': '池子配置',
    'config.section_capacity': '池子容量',
    'config.max_size': '最大容量',
    'config.max_size_help': '代理池总槽位数',
    'config.http_ratio': 'HTTP占比',
    'config.http_ratio_help': '0.5 = 50% HTTP, 50% SOCKS5',
    'config.min_per_protocol': '每协议最小数',
    'config.min_per_protocol_help': '最小保证数量',
    'config.section_latency': '延迟标准 (ms)',
    'config.latency_standard': '标准模式',
    'config.latency_healthy': '健康模式',
    'config.latency_emergency': '紧急模式',
    'config.section_validation': '验证与健康检查',
    'config.validate_concurrency': '验证并发数',
    'config.validate_timeout': '验证超时(秒)',
    'config.health_interval': '检查间隔(分钟)',
    'config.health_batch': '每批数量',
    'config.section_optimization': '优化设置',
    'config.optimize_interval': '优化间隔(分钟)',
    'config.replace_threshold': '替换阈值',
    'config.replace_threshold_help': '新代理需快30%',
    'config.section_geo_filter': '地理过滤',
    'config.allowed_countries': '允许国家（白名单）',
    'config.allowed_countries_help': '非空时仅允许这些国家入池，忽略黑名单',
    'config.blocked_countries': '屏蔽国家（黑名单）',
    'config.blocked_countries_help': '白名单为空时生效',
    'config.cancel': '取消',
    'config.save': '保存配置',
    'msg.fetch_confirm': '确定开始抓取代理吗？',
    'msg.fetch_started': '抓取已在后台启动',
    'msg.refresh_confirm': '确定刷新所有代理的延迟吗？这可能需要一些时间。',
    'msg.refresh_started': '延迟刷新已启动',
    'msg.delete_confirm': '确定删除代理',
    'msg.config_saved': '配置保存成功',
    'msg.config_failed': '配置保存失败',
    // 设置弹窗新增
    'config.system_title': '系统设置',
    'config.section_proxy_mode': '代理使用模式',
    'config.proxy_strategy': '出站代理选择策略',
    'config.mode_mixed_custom': '混合 · 订阅优先（有订阅代理时优先使用，无可用则降级到免费）',
    'config.mode_mixed_free': '混合 · 免费优先（有免费代理时优先使用，无可用则降级到订阅）',
    'config.mode_mixed': '混合 · 平等（不区分来源，按延迟/随机选择）',
    'config.mode_custom_only': '仅订阅代理（只使用订阅导入的代理）',
    'config.mode_free_only': '仅免费代理（只使用公开抓取的代理）',
    'config.section_free_pool': '免费代理池',
    'config.pool_capacity': '池子容量',
    'config.pool_capacity_help': '免费代理总槽位',
    'config.http_ratio_label': 'HTTP 占比',
    'config.latency_standard': '标准延迟 (ms)',
    'config.latency_healthy': '健康延迟 (ms)',
    'config.latency_emergency': '紧急延迟 (ms)',
    'config.section_sub_pool': '订阅代理池',
    'config.probe_interval': '探测间隔 (分钟)',
    'config.probe_interval_help': '禁用代理的唤醒探测间隔',
    'config.refresh_interval': '默认刷新间隔 (分钟)',
    'config.refresh_interval_help': '新订阅的默认刷新周期',
    'config.geo_filter_help': '免费代理删除，订阅代理禁用',
    // 健康面板
    'health.free_pool': '免费池',
    'health.sub_pool': '订阅池',
    'health.free_proxies': '免费代理',
    'health.sub_sources': '订阅源',
    'health.available': '可用',
    'health.disabled': '禁用/待恢复',
    'health.awaiting_probe': '等待探测唤醒',
    'health.no_disabled': '无禁用节点',
    'health.singbox_running': 'sing-box 运行中',
    'health.ready': '就绪',
    'health.not_added': '未添加',
    'health.total_nodes': '共 {0} 节点',
    // 订阅面板
    'sub.title': '订阅管理',
    'sub.add': '添加订阅',
    'sub.refresh_all': '刷新所有订阅',
    'sub.empty': '暂无订阅',
    'sub.nodes': '节点',
    'sub.available': '可用',
    'sub.disabled_label': '禁用',
    'sub.contributed': '贡献',
    // 添加订阅弹窗
    'sub.add_title': '添加订阅',
    'sub.name': '名称',
    'sub.import_mode': '导入方式',
    'sub.tab_url': '订阅 URL',
    'sub.tab_file': '上传文件',
    'sub.url_label': '订阅 URL',
    'sub.url_help': '自动识别格式：Clash YAML / V2ray 链接 / Base64 / 纯文本',
    'sub.file_label': '配置文件',
    'sub.file_drop': '点击选择或拖拽文件到此处',
    'sub.file_formats': '支持 Clash YAML / V2ray 订阅 / 纯文本',
    'sub.refresh_min': '刷新间隔 (分钟)',
    'sub.refresh_min_help': '仅 URL 模式有效，上传文件不自动刷新',
    'sub.cancel': '取消',
    'sub.submit': '添加',
    // 贡献订阅弹窗
    'contribute.title': '贡献订阅',
    'contribute.desc': '分享你的代理订阅，帮助丰富代理池。',
    'contribute.privacy': '你的订阅仅用于此代理池，不会被用于其他渠道。连续探测无可用节点将自动移除。',
    'contribute.submit': '提交',
    'contribute.validating': '验证中...',
    'contribute.nav': '贡献订阅',
    'contribute.settings': '系统设置',
    // 消息
    'msg.sub_added': '订阅已添加，正在导入节点...',
    'msg.sub_refreshed': '刷新已启动',
    'msg.sub_refresh_all': '所有订阅刷新已启动',
    'msg.sub_delete_confirm': '确定删除此订阅？',
    'msg.sub_url_required': '请填写订阅 URL',
    'msg.sub_file_required': '请选择或拖拽配置文件',
    'msg.contribute_thanks': '感谢贡献！订阅已添加，正在导入节点...',
    'msg.submit_failed': '提交失败: ',
  },
  en: {
    'control.title': 'Control Center',
    'nav.config': 'Config',
    'nav.login': 'Login',
    'nav.logout': 'Logout',
    'health.status': 'Pool Status',
    'health.total': 'Total Proxies',
    'health.capacity': 'capacity',
    'health.slots': 'slots',
    'health.avg': 'avg',
    'health.state.healthy': 'Healthy',
    'health.state.warning': 'Warning',
    'health.state.critical': 'Critical',
    'health.state.emergency': 'Emergency',
    'quality.title': 'Quality Distribution',
    'quality.grade_s': 'S Grade',
    'quality.grade_a': 'A Grade',
    'quality.grade_b': 'B Grade',
    'quality.grade_c': 'C Grade',
    'actions.fetch': 'Fetch Proxies',
    'actions.refresh': 'Refresh Latency',
    'actions.config': 'Configure Pool',
    'proxy.title': 'Proxy Registry',
    'proxy.tab_all': 'All',
    'proxy.filter_protocol': 'Protocol',
    'proxy.filter_country': 'Exit Country',
    'proxy.loading': 'Loading...',
    'proxy.empty': 'No proxies available',
    'proxy.th_grade': 'Grade',
    'proxy.th_protocol': 'Protocol',
    'proxy.th_address': 'Address',
    'proxy.th_exit_ip': 'Exit IP',
    'proxy.th_location': 'Location',
    'proxy.th_latency': 'Latency',
    'proxy.th_usage': 'Usage',
    'proxy.th_action': 'Action',
    'proxy.btn_delete': 'DEL',
    'proxy.btn_refresh': 'Refresh',
    'proxy.copy_success': 'Copied',
    'proxy.refresh_started': 'Refresh started',
    'log.title': 'System Log',
    'log.auto_refresh_label': 'Auto Refresh',
    'log.loading': 'Loading...',
    'log.empty': 'No logs',
    'config.title': 'Pool Configuration',
    'config.section_capacity': 'Pool Capacity',
    'config.max_size': 'Max Size',
    'config.max_size_help': 'Total proxy slots',
    'config.http_ratio': 'HTTP Ratio',
    'config.http_ratio_help': '0.5 = 50% HTTP, 50% SOCKS5',
    'config.min_per_protocol': 'Min Per Protocol',
    'config.min_per_protocol_help': 'Minimum guarantee',
    'config.section_latency': 'Latency Standards (ms)',
    'config.latency_standard': 'Standard',
    'config.latency_healthy': 'Healthy',
    'config.latency_emergency': 'Emergency',
    'config.section_validation': 'Validation & Health Check',
    'config.validate_concurrency': 'Validate Concurrency',
    'config.validate_timeout': 'Validate Timeout (s)',
    'config.health_interval': 'Health Check Interval (min)',
    'config.health_batch': 'Batch Size',
    'config.section_optimization': 'Optimization',
    'config.optimize_interval': 'Optimize Interval (min)',
    'config.replace_threshold': 'Replace Threshold',
    'config.replace_threshold_help': 'New proxy must be 30% faster',
    'config.section_geo_filter': 'Geo Filter',
    'config.allowed_countries': 'Allowed Countries (Whitelist)',
    'config.allowed_countries_help': 'When set, only these countries are allowed; blacklist is ignored',
    'config.blocked_countries': 'Blocked Countries (Blacklist)',
    'config.blocked_countries_help': 'Effective only when whitelist is empty',
    'config.cancel': 'Cancel',
    'config.save': 'Save Configuration',
    'msg.fetch_confirm': 'Start proxy fetch?',
    'msg.fetch_started': 'Fetch started in background',
    'msg.refresh_confirm': 'Refresh latency for all proxies? This may take a while.',
    'msg.refresh_started': 'Latency refresh started',
    'msg.delete_confirm': 'Delete proxy',
    'msg.config_saved': 'Configuration saved successfully',
    'msg.config_failed': 'Failed to save configuration',
    'config.system_title': 'System Settings',
    'config.section_proxy_mode': 'Proxy Mode',
    'config.proxy_strategy': 'Outbound Proxy Strategy',
    'config.mode_mixed_custom': 'Mixed · Subscription Priority',
    'config.mode_mixed_free': 'Mixed · Free Priority',
    'config.mode_mixed': 'Mixed · Equal (select by latency/random)',
    'config.mode_custom_only': 'Subscription Only',
    'config.mode_free_only': 'Free Only',
    'config.section_free_pool': 'Free Proxy Pool',
    'config.pool_capacity': 'Pool Capacity',
    'config.pool_capacity_help': 'Total free proxy slots',
    'config.http_ratio_label': 'HTTP Ratio',
    'config.latency_standard': 'Standard Latency (ms)',
    'config.latency_healthy': 'Healthy Latency (ms)',
    'config.latency_emergency': 'Emergency Latency (ms)',
    'config.section_sub_pool': 'Subscription Pool',
    'config.probe_interval': 'Probe Interval (min)',
    'config.probe_interval_help': 'Wake-up probe interval for disabled proxies',
    'config.refresh_interval': 'Default Refresh (min)',
    'config.refresh_interval_help': 'Default refresh cycle for new subscriptions',
    'config.geo_filter_help': 'Free: delete, Subscription: disable',
    'health.free_pool': 'Free Pool',
    'health.sub_pool': 'Subscription Pool',
    'health.free_proxies': 'Free Proxies',
    'health.sub_sources': 'Sources',
    'health.available': 'Available',
    'health.disabled': 'Disabled',
    'health.awaiting_probe': 'Awaiting probe',
    'health.no_disabled': 'No disabled nodes',
    'health.singbox_running': 'sing-box running',
    'health.ready': 'Ready',
    'health.not_added': 'None',
    'health.total_nodes': '{0} total nodes',
    'sub.title': 'Subscriptions',
    'sub.add': 'Add Subscription',
    'sub.refresh_all': 'Refresh All',
    'sub.empty': 'No subscriptions',
    'sub.nodes': 'nodes',
    'sub.available': 'available',
    'sub.disabled_label': 'disabled',
    'sub.contributed': 'Contributed',
    'sub.add_title': 'Add Subscription',
    'sub.name': 'Name',
    'sub.import_mode': 'Import Mode',
    'sub.tab_url': 'URL',
    'sub.tab_file': 'Upload File',
    'sub.url_label': 'Subscription URL',
    'sub.url_help': 'Auto-detect: Clash YAML / V2ray / Base64 / Plain text',
    'sub.file_label': 'Config File',
    'sub.file_drop': 'Click or drag file here',
    'sub.file_formats': 'Supports Clash YAML / V2ray / Plain text',
    'sub.refresh_min': 'Refresh Interval (min)',
    'sub.refresh_min_help': 'URL mode only; file uploads do not auto-refresh',
    'sub.cancel': 'Cancel',
    'sub.submit': 'Add',
    'contribute.title': 'Contribute Subscription',
    'contribute.desc': 'Share your proxy subscription to enrich the pool.',
    'contribute.privacy': 'Your subscription is only used for this proxy pool. Subscriptions with no available nodes for 7 days will be auto-removed.',
    'contribute.submit': 'Submit',
    'contribute.validating': 'Validating...',
    'contribute.nav': 'Contribute',
    'contribute.settings': 'Settings',
    'msg.sub_added': 'Subscription added, importing nodes...',
    'msg.sub_refreshed': 'Refresh started',
    'msg.sub_refresh_all': 'Refreshing all subscriptions',
    'msg.sub_delete_confirm': 'Delete this subscription?',
    'msg.sub_url_required': 'Please enter subscription URL',
    'msg.sub_file_required': 'Please select or drag a config file',
    'msg.contribute_thanks': 'Thanks! Subscription added, importing nodes...',
    'msg.submit_failed': 'Submit failed: ',
  }
};

let currentLang = 'zh';
let logCountdown = 5;

function t(key) {
  return i18n[currentLang][key] || key;
}

function updateLogCountdown() {
  const el = document.getElementById('log-countdown');
  if (el) el.textContent = logCountdown;
}

function updateI18n() {
  document.querySelectorAll('[data-i18n]').forEach(el => {
    const key = el.getAttribute('data-i18n');
    el.textContent = t(key);
  });
  // 更新 title 属性
  document.querySelectorAll('[data-i18n-title]').forEach(el => {
    const key = el.getAttribute('data-i18n-title');
    el.title = t(key);
  });
  document.getElementById('lang-btn').textContent = currentLang === 'zh' ? 'EN' : '中';
  document.title = currentLang === 'zh' ? 'ProxyGate — 智能代理网关' : 'ProxyGate — Intelligent Gateway';

  // 更新筛选下拉框标签
  const protocolLabel = document.getElementById('protocol-filter-label');
  if (protocolLabel) protocolLabel.textContent = t('proxy.filter_protocol');
  const countryLabel = document.getElementById('country-filter-label');
  if (countryLabel) countryLabel.textContent = t('proxy.filter_country');
}

function toggleLang() {
  currentLang = currentLang === 'zh' ? 'en' : 'zh';
  document.getElementById('lang-btn').textContent = currentLang === 'zh' ? 'EN' : '中';
  localStorage.setItem('lang', currentLang);
  updateI18n();
  if (allProxies.length > 0) {
    filterAndRender();
  }
  // 重新渲染包含动态 t() 文字的模块
  loadSubscriptions();
  loadPoolStatus();
}

// 页面加载时恢复语言设置
const savedLang = localStorage.getItem('lang');
if (savedLang) {
  currentLang = savedLang;
  updateI18n();
}

let currentProtocol = '';
let currentCountry = '';
let allProxies = [];
let isAdmin = false; // 是否为管理员

async function api(path, opts) {
  const r = await fetch(path, opts);
  if (r.status === 401) { location.href = '/login'; return null; }
  return r.json();
}

// 检查当前用户权限
async function checkAuth() {
  try {
    const auth = await fetch('/api/auth/check').then(r => r.json());
    isAdmin = auth.isAdmin || false;
    updateUIByRole();
  } catch (e) {
    isAdmin = false;
    updateUIByRole();
  }
}

// 根据角色更新 UI
function updateUIByRole() {
  // 显示/隐藏管理员专属元素
  document.querySelectorAll('.admin-only').forEach(el => {
    if (isAdmin) {
      el.style.display = el.classList.contains('tab') ? 'inline-flex' : 'block';
    } else {
      el.style.display = 'none';
    }
  });
  
  // 显示/隐藏登录链接和访客专属元素
  const loginLink = document.getElementById('login-link');
  if (loginLink) loginLink.style.display = isAdmin ? 'none' : 'inline-flex';
  document.querySelectorAll('.guest-only').forEach(el => {
    el.style.display = isAdmin ? 'none' : 'inline-flex';
  });
  
  // 更新用户模式标识
  const modeEl = document.getElementById('user-mode');
  if (modeEl) {
    if (isAdmin) {
      modeEl.textContent = 'Admin';
    } else {
      modeEl.textContent = 'Guest';
    }
  }
  
  // 重新渲染代理列表（更新操作列）
  if (allProxies.length > 0) {
    filterAndRender();
  }
}

function getCountryFlag(countryCode) {
  if (!countryCode || countryCode === 'UNKNOWN') return '';
  const offset = 127397;
  return countryCode.toUpperCase().split('').map(c => String.fromCodePoint(c.charCodeAt(0) + offset)).join('');
}

function showToast(message) {
  const toast = document.getElementById('toast');
  toast.textContent = message;
  toast.classList.add('show');
  setTimeout(() => toast.classList.remove('show'), 2000);
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => {
    showToast(t('proxy.copy_success') + ': ' + text);
  }).catch(err => {
    console.error('Copy failed:', err);
  });
}

async function refreshProxy(address) {
  const res = await api('/api/proxy/refresh', { address });
  if (res) {
    showToast(t('proxy.refresh_started'));
    setTimeout(() => loadProxies(), 2000);
  }
}

async function loadPoolStatus() {
  const status = await api('/api/pool/status');
  if (!status) return;

  const freeTotal = status.Total - (status.CustomCount || 0);
  document.getElementById('stat-total').textContent = freeTotal;
  document.getElementById('stat-capacity').textContent = status.HTTPSlots + status.SOCKS5Slots;
  document.getElementById('stat-http').textContent = status.HTTP;
  document.getElementById('stat-socks5').textContent = status.SOCKS5;
  document.getElementById('http-slots').textContent = status.HTTPSlots;
  document.getElementById('socks5-slots').textContent = status.SOCKS5Slots;
  document.getElementById('http-avg').textContent = status.AvgLatencyHTTP || '—';
  document.getElementById('socks5-avg').textContent = status.AvgLatencySocks5 || '—';
  
  const stateEl = document.getElementById('pool-state');
  const dotEl = document.getElementById('pool-status-dot');
  const stateText = t('health.state.' + status.State.toLowerCase());
  stateEl.textContent = stateText.toUpperCase();
  dotEl.className = 'health-status ' + status.State.toLowerCase();
}

async function loadQualityDistribution() {
  const dist = await api('/api/pool/quality');
  if (!dist) return;

  const total = (dist.S || 0) + (dist.A || 0) + (dist.B || 0) + (dist.C || 0);
  
  document.getElementById('grade-s-count').textContent = dist.S || 0;
  document.getElementById('grade-a-count').textContent = dist.A || 0;
  document.getElementById('grade-b-count').textContent = dist.B || 0;
  document.getElementById('grade-c-count').textContent = dist.C || 0;

  if (total > 0) {
    const visual = document.getElementById('quality-visual');
    visual.innerHTML = '';
    if (dist.S) visual.innerHTML += '<div class="quality-segment quality-s" style="width:' + (dist.S/total*100) + '%">' + (dist.S/total*100 >= 10 ? 'S' : '') + '</div>';
    if (dist.A) visual.innerHTML += '<div class="quality-segment quality-a" style="width:' + (dist.A/total*100) + '%">' + (dist.A/total*100 >= 10 ? 'A' : '') + '</div>';
    if (dist.B) visual.innerHTML += '<div class="quality-segment quality-b" style="width:' + (dist.B/total*100) + '%">' + (dist.B/total*100 >= 10 ? 'B' : '') + '</div>';
    if (dist.C) visual.innerHTML += '<div class="quality-segment quality-c" style="width:' + (dist.C/total*100) + '%">' + (dist.C/total*100 >= 10 ? 'C' : '') + '</div>';
  }
}

let subNameMap = {};
async function loadProxies() {
  // 先加载订阅名称映射
  const subs = await api('/api/subscriptions');
  if (subs) {
    subNameMap = {};
    subs.forEach(s => { subNameMap[s.id] = s.name || t('sub.add_title'); });
  }

  const path = currentProtocol ? '/api/proxies?protocol=' + currentProtocol : '/api/proxies';
  const proxies = await api(path);
  if (!proxies) return;

  allProxies = proxies;
  updateCountryOptions();
  filterAndRender();
}

function updateCountryOptions() {
  const countries = new Set();
  allProxies.forEach(p => {
    if (p.exit_location) {
      const countryCode = p.exit_location.split(' ')[0];
      if (countryCode) countries.add(countryCode);
    }
  });
  
  const select = document.getElementById('country-filter');
  const currentValue = select.value;
  select.innerHTML = '<option value="" id="country-filter-label">' + t('proxy.filter_country') + '</option>';
  Array.from(countries).sort().forEach(code => {
    const flag = getCountryFlag(code);
    select.innerHTML += '<option value="' + code + '">' + flag + ' ' + code + '</option>';
  });
  if (currentValue && countries.has(currentValue)) {
    select.value = currentValue;
  }
}

function filterAndRender() {
  let filtered = allProxies;
  if (currentCountry) {
    filtered = filtered.filter(p => p.exit_location && p.exit_location.startsWith(currentCountry + ' '));
  }
  renderProxies(filtered);
}

function setProtocolFilter(protocol) {
  currentProtocol = protocol;
  loadProxies();
}

function setCountryFilter(country) {
  currentCountry = country;
  filterAndRender();
}

function renderProxies(proxies) {
  let html = '';
  if (proxies.length === 0) {
    html = '<div class="empty" data-i18n="proxy.empty">' + t('proxy.empty') + '</div>';
  } else {
    html = '<table><thead><tr>';
    html += '<th data-i18n="proxy.th_grade">' + t('proxy.th_grade') + '</th>';
    html += '<th data-i18n="proxy.th_protocol">' + t('proxy.th_protocol') + '</th>';
    html += '<th data-i18n="proxy.th_address">' + t('proxy.th_address') + '</th>';
    html += '<th data-i18n="proxy.th_exit_ip">' + t('proxy.th_exit_ip') + '</th>';
    html += '<th data-i18n="proxy.th_location">' + t('proxy.th_location') + '</th>';
    html += '<th data-i18n="proxy.th_latency">' + t('proxy.th_latency') + '</th>';
    html += '<th data-i18n="proxy.th_usage">' + t('proxy.th_usage') + '</th>';
    if (isAdmin) {
      html += '<th data-i18n="proxy.th_action">' + t('proxy.th_action') + '</th>';
    }
    html += '</tr></thead><tbody>';

    proxies.forEach(p => {
      const flag = p.exit_location ? getCountryFlag(p.exit_location.split(' ')[0]) : '';
      const grade = (p.quality_grade || 'C').toLowerCase();
      const latencyClass = 'grade-' + grade;

      const rowClass = p.source === 'custom' ? ' class="row-custom"' : '';
      html += '<tr' + rowClass + '>';
      html += '<td class="cell-grade grade-' + grade + '">' + (p.quality_grade || 'C') + '</td>';
      html += '<td><span class="badge badge-' + p.protocol + '">' + p.protocol.toUpperCase() + '</span>';
      if (p.source === 'custom') {
        const subName = subNameMap[p.subscription_id] || t('sub.add_title');
        html += ' <span class="source-badge">' + subName + '</span>';
      }
      html += '</td>';
      html += '<td class="cell-mono cell-clickable" onclick="copyToClipboard(\'' + p.address + '\')" title="Copy">' + p.address + '</td>';
      html += '<td class="cell-mono">' + (p.exit_ip || '—') + '</td>';
      html += '<td>' + flag + ' ' + (p.exit_location || '—') + '</td>';
      html += '<td class="cell-mono ' + latencyClass + '">' + (p.latency > 0 ? p.latency + 'ms' : '—') + '</td>';
      html += '<td class="cell-mono">' + (p.use_count || 0) + ' / ' + (p.success_count || 0) + '</td>';
      
      if (isAdmin) {
        html += '<td>';
        html += '<button class="btn-action" onclick="refreshProxy(\'' + p.address + '\')" data-i18n="proxy.btn_refresh">' + t('proxy.btn_refresh') + '</button>';
        html += '<button class="btn-danger" onclick="deleteProxy(\'' + p.address + '\')" data-i18n="proxy.btn_delete">' + t('proxy.btn_delete') + '</button>';
        html += '</td>';
      }
      
      html += '</tr>';
    });

    html += '</tbody></table>';
  }

  document.getElementById('proxy-table-wrap').innerHTML = html;
}

async function triggerFetch() {
  if (!confirm(t('msg.fetch_confirm'))) return;
  await api('/api/fetch', {method: 'POST'});
  alert(t('msg.fetch_started'));
  setTimeout(loadAll, 2000);
}

async function refreshLatency() {
  if (!confirm(t('msg.refresh_confirm'))) return;
  await api('/api/refresh-latency', {method: 'POST'});
  alert(t('msg.refresh_started'));
  setTimeout(loadAll, 2000);
}

async function deleteProxy(addr) {
  if (!confirm(t('msg.delete_confirm') + ' ' + addr + '?')) return;
  await api('/api/proxy/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({address: addr})
  });
  loadProxies();
}

async function loadLogs() {
  const data = await api('/api/logs');
  if (!data) return;
  
  const box = document.getElementById('logs-box');
  if (!data.lines || data.lines.length === 0) {
    box.innerHTML = '<div class="empty" data-i18n="log.empty">' + t('log.empty') + '</div>';
    return;
  }

  let html = '';
  data.lines.forEach(line => {
    let cls = '';
    if (line.includes('error') || line.includes('failed') || line.includes('❌') || line.includes('失败')) cls = 'error';
    if (line.includes('success') || line.includes('✅') || line.includes('completed') || line.includes('成功')) cls = 'success';
    html += '<div class="log-line ' + cls + '">' + line + '</div>';
  });
  box.innerHTML = html;
  box.scrollTop = box.scrollHeight;
  
  // 重置倒计时
  logCountdown = 5;
  
  // 同时刷新代理列表
  loadProxies();
}

async function openSettings() {
  const cfg = await api('/api/config');
  if (!cfg) return;

  document.getElementById('cfg-pool-size').value = cfg.pool_max_size;
  document.getElementById('cfg-http-ratio').value = cfg.pool_http_ratio;
  document.getElementById('cfg-min-per-protocol').value = cfg.pool_min_per_protocol;
  document.getElementById('cfg-max-latency').value = cfg.max_latency_ms;
  document.getElementById('cfg-max-latency-healthy').value = cfg.max_latency_healthy;
  document.getElementById('cfg-max-latency-emergency').value = cfg.max_latency_emergency;
  document.getElementById('cfg-concurrency').value = cfg.validate_concurrency;
  document.getElementById('cfg-timeout').value = cfg.validate_timeout;
  document.getElementById('cfg-health-interval').value = cfg.health_check_interval;
  document.getElementById('cfg-health-batch').value = cfg.health_check_batch_size;
  document.getElementById('cfg-optimize-interval').value = cfg.optimize_interval;
  document.getElementById('cfg-replace-threshold').value = cfg.replace_threshold;
  document.getElementById('cfg-blocked-countries').value = (cfg.blocked_countries || []).join(',');
  document.getElementById('cfg-allowed-countries').value = (cfg.allowed_countries || []).join(',');
  // 将 mode + priority 映射到5种模式
  const mode = cfg.custom_proxy_mode || 'mixed';
  const customPri = cfg.custom_priority === true;
  const freePri = cfg.custom_free_priority === true;
  let uiMode = 'mixed';
  if (mode === 'custom_only') uiMode = 'custom_only';
  else if (mode === 'free_only') uiMode = 'free_only';
  else if (mode === 'mixed' && customPri) uiMode = 'mixed_custom_priority';
  else if (mode === 'mixed' && freePri) uiMode = 'mixed_free_priority';
  else uiMode = 'mixed';
  document.getElementById('cfg-custom-mode').value = uiMode;
  document.getElementById('cfg-custom-probe').value = cfg.custom_probe_interval || 10;
  document.getElementById('cfg-custom-refresh').value = cfg.custom_refresh_interval || 60;

  document.getElementById('settings-modal').classList.add('show');
}

function closeSettings() {
  document.getElementById('settings-modal').classList.remove('show');
}

async function saveConfig() {
  const cfg = {
    pool_max_size: parseInt(document.getElementById('cfg-pool-size').value),
    pool_http_ratio: parseFloat(document.getElementById('cfg-http-ratio').value),
    pool_min_per_protocol: parseInt(document.getElementById('cfg-min-per-protocol').value),
    max_latency_ms: parseInt(document.getElementById('cfg-max-latency').value),
    max_latency_healthy: parseInt(document.getElementById('cfg-max-latency-healthy').value),
    max_latency_emergency: parseInt(document.getElementById('cfg-max-latency-emergency').value),
    validate_concurrency: parseInt(document.getElementById('cfg-concurrency').value),
    validate_timeout: parseInt(document.getElementById('cfg-timeout').value),
    health_check_interval: parseInt(document.getElementById('cfg-health-interval').value),
    health_check_batch_size: parseInt(document.getElementById('cfg-health-batch').value),
    optimize_interval: parseInt(document.getElementById('cfg-optimize-interval').value),
    replace_threshold: parseFloat(document.getElementById('cfg-replace-threshold').value),
    blocked_countries: document.getElementById('cfg-blocked-countries').value.split(',').map(s => s.trim().toUpperCase()).filter(s => s),
    allowed_countries: document.getElementById('cfg-allowed-countries').value.split(',').map(s => s.trim().toUpperCase()).filter(s => s),
    custom_proxy_mode: (() => {
      const m = document.getElementById('cfg-custom-mode').value;
      if (m === 'custom_only') return 'custom_only';
      if (m === 'free_only') return 'free_only';
      return 'mixed';
    })(),
    custom_priority: (() => {
      const m = document.getElementById('cfg-custom-mode').value;
      if (m === 'mixed_custom_priority') return true;
      if (m === 'mixed_free_priority') return false;
      return false;
    })(),
    custom_free_priority: document.getElementById('cfg-custom-mode').value === 'mixed_free_priority',
    custom_probe_interval: parseInt(document.getElementById('cfg-custom-probe').value),
    custom_refresh_interval: parseInt(document.getElementById('cfg-custom-refresh').value),
  };

  const result = await api('/api/config/save', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(cfg)
  });

  if (result && result.status === 'saved') {
    alert(t('msg.config_saved'));
    closeSettings();
    loadAll();
  } else {
    alert(t('msg.config_failed'));
  }
}

async function loadAll() {
  await checkAuth(); // 先检查权限
  loadPoolStatus();
  loadQualityDistribution();
  loadProxies();
  loadLogs();
}

// ========== 订阅管理 ==========

async function loadSubscriptions() {
  const subs = await api('/api/subscriptions');
  const el = document.getElementById('sub-list');
  if (!el || !subs) return;

  if (subs.length === 0) {
    el.innerHTML = '<div class="subscription-empty">' + t('sub.empty') + '</div>';
    return;
  }

  el.innerHTML = subs.map(s => {
    const statusIcon = s.status === 'active' ? '●' : '○';
    const statusClass = s.status === 'active' ? 'subscription-status active' : 'subscription-status inactive';
    const active = s.active_count || 0;
    const disabled = s.disabled_count || 0;
    const total = active + disabled;
    const statsText = total + ' ' + t('sub.nodes') + ' · ' + active + ' ' + t('sub.available') + (disabled > 0 ? ' · ' + disabled + ' ' + t('sub.disabled_label') : '');
    const badge = s.contributed ? '<span class="subscription-badge subscription-badge-warm">' + t('sub.contributed') + '</span>' : '';
    return '<div class="subscription-item">' +
      '<div class="subscription-item-main">' +
        '<div class="subscription-name-row">' +
          '<span class="' + statusClass + '">' + statusIcon + '</span>' +
          '<span class="subscription-name">' + (s.name||t('sub.add_title')) + '</span>' + badge +
        '</div>' +
        '<div class="subscription-stats">' + statsText + '</div>' +
      '</div>' +
      '<div class="subscription-actions">' +
        '<button class="icon-btn" onclick="refreshSub(' + s.id + ')">↻</button>' +
        '<button class="icon-btn" onclick="toggleSub(' + s.id + ')">' + (s.status === 'active' ? '⏸' : '▶') + '</button>' +
        '<button class="icon-btn icon-btn-danger" onclick="deleteSub(' + s.id + ')">✕</button>' +
      '</div>' +
    '</div>';
  }).join('');

  // 加载状态
  const status = await api('/api/custom/status');
  const statusEl = document.getElementById('sub-status');
  if (status && statusEl) {
    const parts = [];
    if (status.singbox_running) parts.push('sing-box ✅ ' + status.singbox_nodes + ' ' + t('sub.nodes'));
    statusEl.textContent = parts.length > 0 ? parts.join(' · ') : '';
  }

  // 更新订阅代理统计卡片
  if (status) {
    const active = status.custom_count || 0;
    const disabled = status.disabled_count || 0;
    const subCount = status.subscription_count || 0;

    const subCountEl = document.getElementById('stat-sub-count');
    const subMetaEl = document.getElementById('stat-sub-meta');
    if (subCountEl) subCountEl.textContent = subCount;
    if (subMetaEl) subMetaEl.textContent = status.singbox_running ? t('health.singbox_running') : (subCount > 0 ? t('health.ready') : t('health.not_added'));

    const customEl = document.getElementById('stat-custom');
    const customMeta = document.getElementById('custom-meta');
    if (customEl) customEl.textContent = active;
    if (customMeta) customMeta.textContent = (active + disabled) > 0 ? t('health.total_nodes').replace('{0}', active + disabled) : '—';

    const disabledEl = document.getElementById('stat-custom-disabled');
    const disabledMeta = document.getElementById('custom-disabled-meta');
    if (disabledEl) disabledEl.textContent = disabled;
    if (disabledMeta) disabledMeta.textContent = disabled > 0 ? t('health.awaiting_probe') : t('health.no_disabled');
  }
}

let subFileContent = '';
let subTab = 'url';

function switchSubTab(tab) {
  subTab = tab;
  document.getElementById('sub-url-group').style.display = tab === 'url' ? '' : 'none';
  document.getElementById('sub-file-group').style.display = tab === 'file' ? '' : 'none';
  document.getElementById('tab-url').className = tab === 'url' ? 'ctrl-btn-primary' : 'ctrl-btn-secondary';
  document.getElementById('tab-file').className = tab === 'file' ? 'ctrl-btn-primary' : 'ctrl-btn-secondary';
}

function handleFileSelect(input) {
  if (input.files && input.files[0]) readSubFile(input.files[0]);
}

function handleFileDrop(e) {
  if (e.dataTransfer.files && e.dataTransfer.files[0]) readSubFile(e.dataTransfer.files[0]);
}

function readSubFile(file) {
  const reader = new FileReader();
  reader.onload = function(e) {
    subFileContent = e.target.result;
    document.getElementById('sub-file-label').innerHTML =
      '<span class="file-selected">✅ ' + file.name + '</span><br>' +
      '<span class="file-selected-meta">' + (subFileContent.length / 1024).toFixed(1) + ' KB</span>';
  };
  reader.readAsText(file);
}

function openSubModal() {
  subFileContent = '';
  subTab = 'url';
  switchSubTab('url');
  document.getElementById('sub-modal').style.display = 'flex';
}

function closeSubModal() {
  document.getElementById('sub-modal').style.display = 'none';
}

async function addSubscription() {
  const name = document.getElementById('sub-name').value || t('sub.add_title');
  const url = document.getElementById('sub-url').value;
  const refreshMin = parseInt(document.getElementById('sub-refresh').value) || 60;

  const data = { name, refresh_min: refreshMin };

  if (subTab === 'url') {
    if (!url) { alert(t('msg.sub_url_required')); return; }
    data.url = url;
  } else {
    if (!subFileContent) { alert(t('msg.sub_file_required')); return; }
    data.file_content = subFileContent;
  }

  const result = await api('/api/subscription/add', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  });

  if (result && result.error) {
    alert(t('msg.submit_failed') + result.error);
    return;
  }
  if (result && result.status === 'added') {
    closeSubModal();
    showToast(t('msg.sub_added'));
    document.getElementById('sub-name').value = '';
    document.getElementById('sub-url').value = '';
    subFileContent = '';
    document.getElementById('sub-file-label').innerHTML = '' + t('sub.file_drop') + '';
    setTimeout(loadSubscriptions, 3000);
    setTimeout(loadProxies, 5000);
  }
}

async function refreshSub(id) {
  await api('/api/subscription/refresh', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({id: id})
  });
  showToast(t('msg.sub_refreshed'));
  setTimeout(loadSubscriptions, 3000);
}

async function refreshAllSubs() {
  await api('/api/subscription/refresh-all', {method: 'POST'});
  showToast(t('msg.sub_refresh_all'));
  setTimeout(loadSubscriptions, 3000);
}

async function toggleSub(id) {
  await api('/api/subscription/toggle', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({id: id})
  });
  loadSubscriptions();
}

async function deleteSub(id) {
  if (!confirm(t('msg.sub_delete_confirm'))) return;
  await api('/api/subscription/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({id: id})
  });
  loadSubscriptions();
}

// ========== 访客贡献订阅 ==========

let contributeFileContent = '';
let contributeTab = 'url';

function switchContributeTab(tab) {
  contributeTab = tab;
  document.getElementById('contribute-url-group').style.display = tab === 'url' ? '' : 'none';
  document.getElementById('contribute-file-group').style.display = tab === 'file' ? '' : 'none';
  document.getElementById('ctab-url').className = tab === 'url' ? 'ctrl-btn-primary' : 'ctrl-btn-secondary';
  document.getElementById('ctab-file').className = tab === 'file' ? 'ctrl-btn-primary' : 'ctrl-btn-secondary';
}

function handleContributeFileSelect(input) {
  if (input.files && input.files[0]) readContributeFile(input.files[0]);
}
function handleContributeFileDrop(e) {
  if (e.dataTransfer.files && e.dataTransfer.files[0]) readContributeFile(e.dataTransfer.files[0]);
}
function readContributeFile(file) {
  const reader = new FileReader();
  reader.onload = function(e) {
    contributeFileContent = e.target.result;
    document.getElementById('contribute-file-label').innerHTML =
      '<span class="file-selected">✅ ' + file.name + '</span><br>' +
      '<span class="file-selected-meta">' + (contributeFileContent.length / 1024).toFixed(1) + ' KB</span>';
  };
  reader.readAsText(file);
}

function openContributeModal() {
  contributeFileContent = '';
  contributeTab = 'url';
  switchContributeTab('url');
  document.getElementById('contribute-modal').style.display = 'flex';
}

function closeContributeModal() {
  document.getElementById('contribute-modal').style.display = 'none';
}

async function submitContribution() {
  const name = document.getElementById('contribute-name').value || t('contribute.title');
  const data = { name };

  if (contributeTab === 'url') {
    const url = document.getElementById('contribute-url').value;
    if (!url) { alert(t('msg.sub_url_required')); return; }
    data.url = url;
  } else {
    if (!contributeFileContent) { alert(t('msg.sub_file_required')); return; }
    data.file_content = contributeFileContent;
  }

  const btn = document.getElementById('contribute-submit-btn');
  btn.textContent = t('contribute.validating');
  btn.disabled = true;

  const result = await api('/api/subscription/contribute', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  });

  btn.textContent = t('contribute.submit');
  btn.disabled = false;

  if (result && result.error) {
    alert(t('msg.submit_failed') + result.error);
    return;
  }
  if (result && result.status === 'contributed') {
    closeContributeModal();
    showToast(t('msg.contribute_thanks'));
    document.getElementById('contribute-name').value = '';
    document.getElementById('contribute-url').value = '';
    contributeFileContent = '';
    document.getElementById('contribute-file-label').innerHTML = '' + t('sub.file_drop') + '';
    setTimeout(loadSubscriptions, 3000);
  }
}

loadAll();
loadSubscriptions();
setInterval(loadPoolStatus, 5000);
setInterval(loadQualityDistribution, 10000);
setInterval(loadLogs, 5000);
setInterval(loadSubscriptions, 30000);

// 日志倒计时
setInterval(() => {
  logCountdown--;
  if (logCountdown < 0) logCountdown = 5;
  updateLogCountdown();
}, 1000);
</script>

<div id="toast" class="toast"></div>

</body>
</html>`
