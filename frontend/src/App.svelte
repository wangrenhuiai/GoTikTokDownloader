<script>
import { onMount } from 'svelte';

let tab = 'status';
let info = null, status = null, settings = null;
let events = [];
let loginState = '?';
let mainState = null, searchState = null;
let keyword = 'football';
let dbResult = '';

function push(tag, data) {
  events = [...events.slice(-99), `${new Date().toLocaleTimeString()} [${tag}] ${typeof data === 'string' ? data : JSON.stringify(data)}`];
}

async function refresh() {
  try {
    info = await window.go.app.App.GetAppInfo();
    status = await window.go.app.App.GetRuntimeStatus();
    settings = await window.go.app.App.GetSettings();
    loginState = await window.go.app.App.GetLoginState();
    mainState = await window.go.app.App.GetMainBrowserState();
    searchState = await window.go.app.App.GetSearchBrowserState();
  } catch (e) { push('error', String(e)); }
}

async function ping() {
  const r = await window.go.app.App.PingEvent('hello');
  push('ping', r);
}

async function dbSmoke() {
  try { dbResult = await window.go.app.App.DBSmokeTest(); }
  catch (e) { dbResult = 'FAIL: ' + e; }
}

async function openMain() { mainState = await window.go.app.App.OpenMainBrowser(); }
async function closeMain() { await window.go.app.App.CloseMainBrowser(); mainState = await window.go.app.App.GetMainBrowserState(); }
async function openSearch() { searchState = await window.go.app.App.OpenSearchBrowser(keyword); }
async function closeSearch() { await window.go.app.App.CloseSearchBrowser(); searchState = await window.go.app.App.GetSearchBrowserState(); }
async function openLogin() { await window.go.app.App.OpenLoginWindow(); mainState = await window.go.app.App.GetMainBrowserState(); }
async function confirmLogin() { loginState = await window.go.app.App.ConfirmLoggedIn(); }
async function probeLogin() { loginProbe = await window.go.app.App.ProbeLoginState(); }
let loginProbe = '';
let jsOut = '';
async function runJS(which, script) {
  try { jsOut = which + ' :: ' + script + '\n=> ' + await window.go.app.App.RunScript(which, script); }
  catch (e) { jsOut = 'FAIL: ' + e; }
}
async function pageInfo(which) {
  try { jsOut = which + ' pageInfo\n=> ' + JSON.stringify(await window.go.app.App.PageInfo(which), null, 2); }
  catch (e) { jsOut = 'FAIL: ' + e; }
}

onMount(() => {
  refresh();
  try {
    window.runtime.EventsOn('system:log', (d) => push('system:log', d));
    window.runtime.EventsOn('system:status', (d) => push('system:status', d));
    window.runtime.EventsOn('task:progress', (d) => push('task:progress', d));
    push('events', 'EventsOn subscribed: system:log/system:status/task:progress');
  } catch (e) { push('error', 'EventsOn unavailable: ' + e); }
});
</script>

<div class="top">
  <b>GoTikTokDownloader</b>
  <span class="badge">TikTok: {loginState}</span>
  <span class="badge">{info ? info.version : '...'}</span>
  <button on:click={refresh}>刷新</button>
</div>
<div class="layout">
  <div class="side">
    <button class:active={tab==='status'} on:click={() => tab='status'}>下载任务</button>
    <button class:active={tab==='search'} on:click={() => tab='search'}>关键词搜索</button>
    <button class:active={tab==='browser'} on:click={() => tab='browser'}>TikTok 浏览器</button>
    <button class:active={tab==='settings'} on:click={() => tab='settings'}>设置</button>
  </div>
  <div class="main">
    {#if tab==='status'}
      <div class="card"><h3>Runtime 状态（来自 Go）</h3>
        <pre>{JSON.stringify({info, status}, null, 2)}</pre>
        <div class="row"><button on:click={ping}>测试事件</button><button on:click={dbSmoke}>SQLite 测试</button><span>{dbResult}</span></div>
      </div>
      <div class="card"><h3>事件日志（EventsOn）</h3><pre class="log">{events.join('\n')}</pre></div>
    {:else if tab==='search'}
      <div class="card"><h3>关键词搜索（Phase 2：仅启动搜索浏览器）</h3>
        <div class="row"><input bind:value={keyword} placeholder="football" /><button on:click={openSearch}>搜索</button><button on:click={closeSearch}>关闭搜索浏览器</button></div>
        <pre>{JSON.stringify(searchState, null, 2)}</pre>
      </div>
    {:else if tab==='browser'}
      <div class="card"><h3>TikTok 登录（软件内部浏览器）</h3>
        <div class="row"><button on:click={openLogin}>打开登录窗口</button><button on:click={confirmLogin}>我已扫码成功</button><button on:click={probeLogin}>检测登录会话</button></div>
        <div class="row"><span>状态：{loginState}</span><span>会话探针：{loginProbe}</span></div>
      </div>
      <div class="card"><h3>主浏览器</h3>
        <div class="row"><button on:click={openMain}>打开主浏览器</button><button on:click={closeMain}>关闭主浏览器</button></div>
        <pre>{JSON.stringify(mainState, null, 2)}</pre>
      </div>
      <div class="card"><h3>搜索浏览器</h3>
        <div class="row"><button on:click={openSearch}>打开搜索浏览器</button><button on:click={closeSearch}>关闭搜索浏览器</button></div>
        <pre>{JSON.stringify(searchState, null, 2)}</pre>
      </div>
      <div class="card"><h3>JavaScript 验证（真实 WebView2 回传）</h3>
        <div class="row">
          <button on:click={() => runJS('main', 'document.title')}>main: document.title</button>
          <button on:click={() => runJS('main', 'location.href')}>main: location.href</button>
          <button on:click={() => runJS('main', 'document.body ? document.body.innerText.length : -1')}>main: body 长度</button>
        </div>
        <div class="row">
          <button on:click={() => pageInfo('main')}>main: PageInfo</button>
          <button on:click={() => pageInfo('search')}>search: PageInfo</button>
          <button on:click={() => runJS('search', 'location.href')}>search: location.href</button>
        </div>
        <pre>{jsOut}</pre>
      </div>
    {:else}
      <div class="card"><h3>设置（来自 Go）</h3><pre>{JSON.stringify(settings, null, 2)}</pre></div>
    {/if}
  </div>
</div>
