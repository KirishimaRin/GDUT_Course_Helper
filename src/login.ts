// 登录视图：JSESSIONID 导入
import { importSession, state, errMsg, toastErr } from "./store";
import { h, esc, refreshIcons } from "./ui";

export function renderLogin(root: HTMLElement, expired: boolean) {
  root.innerHTML = `
    <div class="login-wrap">
      <div class="login-card">
        <div class="login-brand">
          <span class="gh-logo"><i data-lucide="graduation-cap"></i></span>
          <div>
            <div class="login-title">广工抢课助手</div>
            <div class="login-sub">桌面版${state.version ? " v" + esc(state.version) : ""}</div>
          </div>
        </div>
        ${expired ? `<div class="login-expired">登录已失效，请重新导入</div>` : ""}
        <input id="jsessionid" class="gh-input mono" type="text" spellcheck="false"
               placeholder="粘贴 JSESSIONID" autocomplete="off" />
        <button id="importBtn" class="gh-btn gh-primary gh-block">
          <span>导入并登录</span>
        </button>
        <details class="login-help">
          <summary>怎么获取 JSESSIONID？</summary>
          <ol>
            <li>浏览器登录教务系统 <span class="mono">jxfw.gdut.edu.cn</span></li>
            <li>按 F12 打开开发者工具 → 应用/Storage → Cookies</li>
            <li>复制 <span class="mono">JSESSIONID</span> 的值粘贴到上面</li>
          </ol>
        </details>
        <div class="login-foot">会话只保存在本机内存，关闭应用即清除</div>
      </div>
    </div>
  `;
  refreshIcons();

  const input = root.querySelector<HTMLInputElement>("#jsessionid")!;
  const btn = root.querySelector<HTMLButtonElement>("#importBtn")!;
  input.focus();

  const submit = async () => {
    const v = input.value.trim();
    if (!v) {
      toastErr("请先粘贴 JSESSIONID");
      return;
    }
    btn.disabled = true;
    (btn.firstElementChild as HTMLElement).textContent = "验证中…";
    try {
      await importSession(v);
    } catch (e) {
      toastErr(errMsg(e));
      btn.disabled = false;
      (btn.firstElementChild as HTMLElement).textContent = "导入并登录";
    }
  };
  btn.addEventListener("click", submit);
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") submit();
  });
}
