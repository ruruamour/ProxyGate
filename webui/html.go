package webui

const loginHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ProxyGate — 身份验证</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&family=Noto+Sans+SC:wght@400;500;700&family=JetBrains+Mono:wght@500;600&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#f2ebd9;
  --bg-soft:#f8f1e4;
  --card:rgba(253,251,247,0.92);
  --card-solid:#fdfbf7;
  --fg:#1c1917;
  --fg-dim:#5f574d;
  --fg-soft:#8a7f72;
  --border:#e8ddd4;
  --border-strong:#d8c8b8;
  --primary:#2d6a4f;
  --primary-strong:#1b4332;
  --primary-soft:#eef5f1;
  --danger:#d97868;
  --danger-soft:#fdf2f1;
  --shadow:0 18px 40px rgba(67,56,42,0.1),0 4px 12px rgba(67,56,42,0.05);
  --shadow-soft:0 8px 24px rgba(67,56,42,0.08);
  --radius:24px;
  --radius-lg:30px;
  --radius-pill:999px;
}
body{
  min-height:100vh;
  display:flex;
  align-items:center;
  justify-content:center;
  padding:32px 20px;
  background:radial-gradient(circle at top,#fbf6ec 0%,var(--bg) 52%,#e9dcc4 100%);
  color:var(--fg);
  font-family:Nunito,Noto Sans SC,system-ui,sans-serif;
  position:relative;
  overflow:hidden;
}
body::before{
  content:'';
  position:fixed;
  inset:auto auto -140px -120px;
  width:420px;
  height:420px;
  border-radius:50%;
  background:radial-gradient(circle,rgba(45,106,79,0.16) 0%,rgba(45,106,79,0) 68%);
  pointer-events:none;
}
body::after{
  content:'';
  position:fixed;
  inset:0;
  background:
    radial-gradient(circle at 88% 16%,rgba(224,177,94,0.16),transparent 22%),
    radial-gradient(circle at 18% 84%,rgba(45,106,79,0.12),transparent 26%);
  pointer-events:none;
}
.shell{
  width:min(980px,100%);
  display:grid;
  grid-template-columns:minmax(260px,1fr) minmax(320px,440px);
  gap:28px;
  align-items:center;
  position:relative;
  z-index:1;
}
.hero{padding:16px 12px 16px 0}
.eyebrow{
  display:inline-flex;
  align-items:center;
  gap:8px;
  padding:8px 14px;
  border-radius:var(--radius-pill);
  background:rgba(253,251,247,0.76);
  border:1px solid rgba(232,221,212,0.9);
  color:var(--primary);
  font-size:11px;
  font-weight:800;
  letter-spacing:0.08em;
  text-transform:uppercase;
  box-shadow:var(--shadow-soft);
}
.hero h1{
  margin:18px 0 12px;
  font-size:clamp(42px,6vw,74px);
  line-height:0.96;
  letter-spacing:-0.05em;
  font-weight:800;
}
.hero p{
  max-width:420px;
  color:var(--fg-dim);
  font-size:15px;
  line-height:1.8;
}
.card{
  position:relative;
  padding:34px;
  background:var(--card);
  border:1px solid rgba(232,221,212,0.95);
  border-radius:var(--radius-lg);
  box-shadow:var(--shadow);
  backdrop-filter:blur(18px);
}
.github{
  position:absolute;
  top:20px;
  right:20px;
  width:42px;
  height:42px;
  display:inline-flex;
  align-items:center;
  justify-content:center;
  border-radius:50%;
  border:1px solid var(--border);
  background:rgba(253,251,247,0.88);
  color:var(--fg-dim);
  transition:all 0.2s ease;
}
.github:hover{
  transform:translateY(-1px);
  color:var(--primary);
  border-color:var(--border-strong);
  box-shadow:var(--shadow-soft);
}
.logo{
  width:56px;
  height:56px;
  display:grid;
  place-items:center;
  margin-bottom:18px;
  border-radius:18px;
  background:linear-gradient(135deg,var(--primary-soft),#f5efe6);
  color:var(--primary);
  font-family:"JetBrains Mono",monospace;
  font-size:18px;
  font-weight:700;
  letter-spacing:0.08em;
}
h1{
  font-size:34px;
  line-height:1.05;
  letter-spacing:-0.03em;
  margin-bottom:8px;
  font-weight:800;
}
.sub{
  color:var(--fg-dim);
  font-size:14px;
  margin-bottom:28px;
}
label{
  display:block;
  margin-bottom:10px;
  color:var(--fg);
  font-size:12px;
  font-weight:700;
}
input[type=password]{
  width:100%;
  height:50px;
  padding:0 16px;
  border-radius:18px;
  border:1px solid var(--border);
  background:var(--bg-soft);
  color:var(--fg);
  font-size:15px;
  outline:none;
  transition:all 0.2s ease;
}
input[type=password]:focus{
  border-color:rgba(45,106,79,0.45);
  background:var(--card-solid);
  box-shadow:0 0 0 4px rgba(45,106,79,0.12);
}
button{
  width:100%;
  height:50px;
  margin-top:18px;
  border:none;
  border-radius:var(--radius-pill);
  background:linear-gradient(135deg,var(--primary),var(--primary-strong));
  color:#fff;
  font-size:14px;
  font-weight:800;
  cursor:pointer;
  transition:all 0.2s ease;
  box-shadow:var(--shadow-soft);
}
button:hover{
  transform:translateY(-1px);
  box-shadow:var(--shadow);
}
.error{
  margin-bottom:20px;
  padding:14px 16px;
  border-radius:18px;
  border:1px solid rgba(217,120,104,0.28);
  background:var(--danger-soft);
  color:#9a4336;
  font-size:13px;
  font-weight:700;
}
.tip{
  margin-top:18px;
  color:var(--fg-soft);
  font-size:12px;
  line-height:1.8;
}
.tip a{
  color:var(--primary);
  text-decoration:none;
  font-weight:700;
}
.tip a:hover{text-decoration:underline}
@media (max-width: 860px) {
  .shell{grid-template-columns:1fr;gap:18px}
  .hero{padding:0}
  .hero h1{font-size:42px}
  .card{padding:28px}
}
</style>
</head>
<body>
<div class="shell">
  <section class="hero">
    <div class="eyebrow">Unified Proxy Gateway</div>
    <h1>ProxyGate</h1>
  </section>
  <div class="card">
    <a href="https://github.com/ruruamour/ProxyGate" target="_blank" class="github" title="GitHub">
      <svg width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
        <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
      </svg>
    </a>
    <div class="logo">PG</div>
    <h1>管理员登录</h1>
    <p class="sub">进入配置、订阅和维护操作界面</p>
    <form method="POST" action="/login">
      <label>访问密码</label>
      <input type="password" name="password" placeholder="请输入管理员密码" autofocus>
      <button type="submit">进入控制台</button>
    </form>
    <p class="tip">访客模式可<a href="/">查看数据</a>，管理员登录后可完全控制。</p>
  </div>
</div>
</body>
</html>`

const loginHTMLWithError = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ProxyGate — 身份验证</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&family=Noto+Sans+SC:wght@400;500;700&family=JetBrains+Mono:wght@500;600&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#f2ebd9;
  --bg-soft:#f8f1e4;
  --card:rgba(253,251,247,0.92);
  --card-solid:#fdfbf7;
  --fg:#1c1917;
  --fg-dim:#5f574d;
  --fg-soft:#8a7f72;
  --border:#e8ddd4;
  --border-strong:#d8c8b8;
  --primary:#2d6a4f;
  --primary-strong:#1b4332;
  --primary-soft:#eef5f1;
  --danger:#d97868;
  --danger-soft:#fdf2f1;
  --shadow:0 18px 40px rgba(67,56,42,0.1),0 4px 12px rgba(67,56,42,0.05);
  --shadow-soft:0 8px 24px rgba(67,56,42,0.08);
  --radius:24px;
  --radius-lg:30px;
  --radius-pill:999px;
}
body{
  min-height:100vh;
  display:flex;
  align-items:center;
  justify-content:center;
  padding:32px 20px;
  background:radial-gradient(circle at top,#fbf6ec 0%,var(--bg) 52%,#e9dcc4 100%);
  color:var(--fg);
  font-family:Nunito,Noto Sans SC,system-ui,sans-serif;
  position:relative;
  overflow:hidden;
}
body::before{
  content:'';
  position:fixed;
  inset:auto auto -140px -120px;
  width:420px;
  height:420px;
  border-radius:50%;
  background:radial-gradient(circle,rgba(45,106,79,0.16) 0%,rgba(45,106,79,0) 68%);
  pointer-events:none;
}
body::after{
  content:'';
  position:fixed;
  inset:0;
  background:
    radial-gradient(circle at 88% 16%,rgba(224,177,94,0.16),transparent 22%),
    radial-gradient(circle at 18% 84%,rgba(45,106,79,0.12),transparent 26%);
  pointer-events:none;
}
.shell{
  width:min(980px,100%);
  display:grid;
  grid-template-columns:minmax(260px,1fr) minmax(320px,440px);
  gap:28px;
  align-items:center;
  position:relative;
  z-index:1;
}
.hero{padding:16px 12px 16px 0}
.eyebrow{
  display:inline-flex;
  align-items:center;
  gap:8px;
  padding:8px 14px;
  border-radius:var(--radius-pill);
  background:rgba(253,251,247,0.76);
  border:1px solid rgba(232,221,212,0.9);
  color:var(--primary);
  font-size:11px;
  font-weight:800;
  letter-spacing:0.08em;
  text-transform:uppercase;
  box-shadow:var(--shadow-soft);
}
.hero h1{
  margin:18px 0 12px;
  font-size:clamp(42px,6vw,74px);
  line-height:0.96;
  letter-spacing:-0.05em;
  font-weight:800;
}
.hero p{
  max-width:420px;
  color:var(--fg-dim);
  font-size:15px;
  line-height:1.8;
}
.card{
  position:relative;
  padding:34px;
  background:var(--card);
  border:1px solid rgba(232,221,212,0.95);
  border-radius:var(--radius-lg);
  box-shadow:var(--shadow);
  backdrop-filter:blur(18px);
}
.github{
  position:absolute;
  top:20px;
  right:20px;
  width:42px;
  height:42px;
  display:inline-flex;
  align-items:center;
  justify-content:center;
  border-radius:50%;
  border:1px solid var(--border);
  background:rgba(253,251,247,0.88);
  color:var(--fg-dim);
  transition:all 0.2s ease;
}
.github:hover{
  transform:translateY(-1px);
  color:var(--primary);
  border-color:var(--border-strong);
  box-shadow:var(--shadow-soft);
}
.logo{
  width:56px;
  height:56px;
  display:grid;
  place-items:center;
  margin-bottom:18px;
  border-radius:18px;
  background:linear-gradient(135deg,var(--primary-soft),#f5efe6);
  color:var(--primary);
  font-family:"JetBrains Mono",monospace;
  font-size:18px;
  font-weight:700;
  letter-spacing:0.08em;
}
h1{
  font-size:34px;
  line-height:1.05;
  letter-spacing:-0.03em;
  margin-bottom:8px;
  font-weight:800;
}
.sub{
  color:var(--fg-dim);
  font-size:14px;
  margin-bottom:28px;
}
label{
  display:block;
  margin-bottom:10px;
  color:var(--fg);
  font-size:12px;
  font-weight:700;
}
input[type=password]{
  width:100%;
  height:50px;
  padding:0 16px;
  border-radius:18px;
  border:1px solid var(--border);
  background:var(--bg-soft);
  color:var(--fg);
  font-size:15px;
  outline:none;
  transition:all 0.2s ease;
}
input[type=password]:focus{
  border-color:rgba(45,106,79,0.45);
  background:var(--card-solid);
  box-shadow:0 0 0 4px rgba(45,106,79,0.12);
}
button{
  width:100%;
  height:50px;
  margin-top:18px;
  border:none;
  border-radius:var(--radius-pill);
  background:linear-gradient(135deg,var(--primary),var(--primary-strong));
  color:#fff;
  font-size:14px;
  font-weight:800;
  cursor:pointer;
  transition:all 0.2s ease;
  box-shadow:var(--shadow-soft);
}
button:hover{
  transform:translateY(-1px);
  box-shadow:var(--shadow);
}
.error{
  margin-bottom:20px;
  padding:14px 16px;
  border-radius:18px;
  border:1px solid rgba(217,120,104,0.28);
  background:var(--danger-soft);
  color:#9a4336;
  font-size:13px;
  font-weight:700;
}
.tip{
  margin-top:18px;
  color:var(--fg-soft);
  font-size:12px;
  line-height:1.8;
}
.tip a{
  color:var(--primary);
  text-decoration:none;
  font-weight:700;
}
.tip a:hover{text-decoration:underline}
@media (max-width: 860px) {
  .shell{grid-template-columns:1fr;gap:18px}
  .hero{padding:0}
  .hero h1{font-size:42px}
  .card{padding:28px}
}
</style>
</head>
<body>
<div class="shell">
  <section class="hero">
    <div class="eyebrow">Unified Proxy Gateway</div>
    <h1>ProxyGate</h1>
    <p>公开代理抓取、订阅导入、验证入池和统一 HTTP/SOCKS5 输出，都在同一个暖色控制台里完成。</p>
  </section>
  <div class="card">
    <a href="https://github.com/ruruamour/ProxyGate" target="_blank" class="github" title="GitHub">
      <svg width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
        <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
      </svg>
    </a>
    <div class="logo">PG</div>
    <h1>管理员登录</h1>
    <p class="sub">进入配置、订阅和维护操作界面</p>
    <div class="error">密码不正确，请重新输入。</div>
    <form method="POST" action="/login">
      <label>访问密码</label>
      <input type="password" name="password" placeholder="请输入管理员密码" autofocus>
      <button type="submit">重新验证</button>
    </form>
    <p class="tip">访客模式可<a href="/">查看数据</a>，管理员登录后可完全控制。</p>
  </div>
</div>
</body>
</html>`

// dashboardHTML 已移至 dashboard.go
