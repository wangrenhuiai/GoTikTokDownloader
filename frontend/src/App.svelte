<script>
import { onMount } from 'svelte';

let tab = 'status';
let info = null, status = null, settings = null;
let events = [];
let loginState = '?';
let browserState = null, legacy = null;
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
    browserState = await window.go.app.App.GetTikTokBrowserState();
    legacy = await window.go.app.App.LegacyProfiles();
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

async function openBrowser() { browserState = await window.go.app.App.OpenTikTokBrowser(); }
async function closeBrowser() { await window.go.app.App.CloseTikTokBrowser(); browserState = await window.go.app.App.GetTikTokBrowserState(); }
async function openSearch() { browserState = await window.go.app.App.OpenTikTokSearch(keyword); }
async function openLogin() { await window.go.app.App.OpenLoginWindow(); browserState = await window.go.app.App.GetTikTokBrowserState(); }
async function confirmLogin() { loginState = await window.go.app.App.ConfirmLoggedIn(); }
async function probeLogin() { loginProbe = await window.go.app.App.ProbeLoginState(); }
async function waitContent() { contentOut = await window.go.app.App.WaitPageReady(30000); }
let loginProbe = '', contentOut = '';
let jsOut = '';
let searchState = 'Idle', searchResults = [], checked = {}, maxResults = 100;
async function runJS(script) {
  try { jsOut = script + '\n=> ' + await window.go.app.App.RunScript(script); }
  catch (e) { jsOut = 'FAIL: ' + e; }
}
async function pageInfo() {
  try { jsOut = 'pageInfo\n=> ' + JSON.stringify(await window.go.app.App.PageInfo(), null, 2); }
  catch (e) { jsOut = 'FAIL: ' + e; }
}
async function startSearch() {
  searchState = await window.go.app.App.StartSearch(keyword, maxResults);
  await pullSearch();
}
async function stopSearch() { searchState = await window.go.app.App.StopSearch(); }
async function pullSearch() {
  try {
    searchState = await window.go.app.App.SearchState();
    searchResults = await window.go.app.App.SearchResults();
  } catch (e) { push('error', String(e)); }
}
async function clearSearch() { searchState = await window.go.app.App.ClearSearch(); searchResults = []; checked = {}; }
function selectAll(v) {
  const o = {};
  for (const c of searchResults) o[c.videoId] = v;
  checked = o;
}

onMount(() => {
  refresh();
  try {
    window.runtime.EventsOn('system:log', (d) => push('system:log', d));
    window.runtime.EventsOn('system:status', (d) => push('system:status', d));
    window.runtime.EventsOn('task:progress', (d) => push('task:progress', d));
    window.runtime.EventsOn('search:started', (d) => { push('search:started', d); pullSearch(); });
    window.runtime.EventsOn('search:candidate', () => pullSearch());
    window.runtime.EventsOn('search:progress', (d) => push('search:progress', d));
    window.runtime.EventsOn('search:completed', (d) => { push('search:completed', d); pullSearch(); });
    window.runtime.EventsOn('search:error', (d) => push('search:error', d));
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
      <div class="card"><h3>关键词搜索（唯一浏览器复用导航）</h3>
        <div class="row">
          <input bind:value={keyword} placeholder="cooking" />
          <input bind:value={maxResults} type="number" style="width:90px" title="MaxResults" />
          <button on:click={startSearch}>搜索</button>
          <button on:click={stopSearch}>停止</button>
          <button on:click={pullSearch}>刷新结果</button>
          <button on:click={clearSearch}>清空结果</button>
        </div>
        <div class="row"><span>搜索状态：{searchState}</span><span>发现：{searchResults.length} 个视频</span></div>
        <div class="row">
          <button on:click={() => selectAll(true)}>全选</button>
          <button on:click={() => selectAll(false)}>反选</button>
          <span>已选：{Object.values(checked).filter(Boolean).length}</span>
        </div>
        <table border="1" cellpadding="4" style="border-collapse:collapse;width:100%;font-size:12px">
          <thead><tr><th>□</th><th>作者</th><th>视频ID</th><th>视频链接</th></tr></thead>
          <tbody>
          {#each searchResults as c}
          <tr>
            <td><input type="checkbox" bind:checked={checked[c.videoId]} /></td>
            <td>{c.authorName}</td>
            <td>{c.videoId}</td>
            <td style="word-break:break-all">{c.url}</td>
          </tr>
          {/each}
          </tbody>
        </table>
      </div>
    {:else if tab==='browser'}
      <div class="card"><h3>TikTok 登录（唯一软件内部浏览器）</h3>
        <div class="row"><button on:click={openLogin}>打开登录页面</button><button on:click={confirmLogin}>我已扫码成功</button><button on:click={probeLogin}>检测登录会话</button><button on:click={waitContent}>等待内容就绪</button></div>
        <div class="row"><span>状态：{loginState}</span><span>会话探针：{loginProbe}</span></div>
        <pre>{contentOut}</pre>
      </div>
      <div class="card"><h3>TikTok 浏览器（唯一实例）</h3>
        <div class="row"><button on:click={openBrowser}>打开浏览器</button><button on:click={closeBrowser}>关闭浏览器</button></div>
        <pre>{JSON.stringify(browserState, null, 2)}</pre>
        <div class="row"><span>旧 profiles（仅报告，不删除）：{JSON.stringify(legacy)}</span></div>
      </div>
      <div class="card"><h3>JavaScript 验证（真实 WebView2 回传）</h3>
        <div class="row">
          <button on:click={() => runJS('document.title')}>document.title</button>
          <button on:click={() => runJS('location.href')}>location.href</button>
          <button on:click={() => runJS('document.body ? document.body.innerText.length : -1')}>body 长度</button>
          <button on:click={pageInfo}>PageInfo</button>
        </div>
        <pre>{jsOut}</pre>
      </div>
    {:else}
      <div class="card"><h3>设置（来自 Go）</h3><pre>{JSON.stringify(settings, null, 2)}</pre></div>
    {/if}
  </div>
</div>
