// 主视图：课程列表 + 侧栏（收藏/已选）+ 抢课控制 + 日志 + 详情弹窗
import {
  state, onRender, courseById, toggleWishlist, moveWishlist, clearWishlist,
  setCategory, setSearch, startGrab, stopGrab, quickGrab, dropCourse,
  fetchSchedule, clearLogs, loadCourses, loadSelected, loadInterval,
  saveInterval, toastOk, toastErr,
} from "./store";
import type { Course } from "./store";
import { h, esc, refreshIcons } from "./ui";
import { App } from "./mygo";

let logOpen = false;

const WEEK_DAYS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];

export function renderMain(root: HTMLElement) {
  root.innerHTML = `
    <div class="gh-app">
      <div class="gh-body">
        <aside class="gh-side">
          <div class="gh-side-head">收藏 <span id="wishCount" class="gh-badge">0</span></div>
          <div class="gh-favs" id="wishList"></div>
          <div class="gh-side-head">已选 <span id="selCount" class="gh-badge">0</span></div>
          <div class="gh-favs gh-selected" id="selList"></div>
          <div class="gh-user">
            <span class="gh-avatar"><i data-lucide="graduation-cap"></i></span>
            <span class="gh-user-info"><b>已登录</b><small>JSESSIONID 会话</small></span>
            <button id="logoutBtn" class="gh-btn gh-ghost gh-xs">登出</button>
          </div>
        </aside>
        <main class="gh-main">
          <div class="gh-toolbar">
            <label class="gh-search">
              <i data-lucide="search"></i>
              <input id="q" type="search" placeholder="搜索课程或教师" />
            </label>
            <div class="gh-chips" id="chips"></div>
            <div class="gh-run-group">
              <label class="gh-interval" title="抢课间隔">
                <input id="interval" type="number" min="1" max="60" />
                <span>秒</span>
              </label>
              <button id="refreshBtn" class="gh-icon-btn" data-tooltip="刷新课程"><i data-lucide="refresh-cw"></i></button>
              <button id="runBtn" class="gh-btn gh-primary"></button>
            </div>
          </div>
          <div class="gh-semi" id="semInfo"></div>
          <div class="gh-list" id="courseList"></div>
        </main>
      </div>
      <footer class="gh-ticker">
        <span class="gh-tick-dot" id="tickDot"></span>
        <span class="gh-tick-msg" id="tickMsg">就绪</span>
        <span class="gh-tick-count" id="tickCount"></span>
        <button id="logToggle" class="gh-icon-btn" aria-label="抢课日志"><i data-lucide="chevron-up"></i></button>
      </footer>
      <div class="gh-logpanel" id="logPanel"></div>
      <div class="gh-modal" id="modal"></div>
    </div>
  `;

  bindToolbar(root);
  bindSidebar(root);
  bindTicker(root);
  bindModal(root);

  onRender(() => {
    renderChips();
    renderSem();
    renderCourses();
    renderWishlist();
    renderSelected();
    renderRunBtn();
    renderTicker();
    renderLogPanel();
    refreshIcons();
  });

  renderAll();
}

function renderAll() {
  renderChips(); renderSem(); renderCourses(); renderWishlist();
  renderSelected(); renderRunBtn(); renderTicker(); renderLogPanel();
  refreshIcons();
}

// ---------- 工具栏 ----------

function bindToolbar(root: HTMLElement) {
  const q = root.querySelector<HTMLInputElement>("#q")!;
  let t: number | undefined;
  q.addEventListener("input", () => {
    clearTimeout(t);
    t = window.setTimeout(() => setSearch(q.value.trim()), 180);
  });

  root.querySelector("#refreshBtn")!.addEventListener("click", () => loadCourses());

  const interval = root.querySelector<HTMLInputElement>("#interval")!;
  interval.value = String(loadInterval());
  interval.addEventListener("change", () => {
    let v = parseInt(interval.value, 10);
    if (!(v >= 1)) v = 1;
    if (v > 60) v = 60;
    interval.value = String(v);
    saveInterval(v);
  });

  root.querySelector("#runBtn")!.addEventListener("click", () => {
    if (state.running) stopGrab();
    else startGrab();
  });
}

function renderChips() {
  const box = document.getElementById("chips")!;
  const counts = new Map<string, number>();
  state.courses.forEach((c) => {
    const cat = c.kcflmc || c.kcdlmc || "";
    if (cat) counts.set(cat, (counts.get(cat) || 0) + 1);
  });
  const cats = [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 7).map(([k]) => k);

  const chips = [{ key: "", label: "全部" }, ...cats.map((k) => ({ key: k, label: k }))];
  box.innerHTML = chips
    .map((ch) => `<button class="gh-chip ${state.category === ch.key ? "is-on" : ""}" data-cat="${esc(ch.key)}">${esc(ch.label)}</button>`)
    .join("");
  box.querySelectorAll(".gh-chip").forEach((el) => {
    el.addEventListener("click", () => setCategory((el as HTMLElement).dataset.cat || ""));
  });
}

function renderRunBtn() {
  const btn = document.getElementById("runBtn")!;
  btn.classList.toggle("is-stop", state.running);
  btn.innerHTML = state.running
    ? `<i data-lucide="square"></i><span>停止抢课</span>`
    : `<i data-lucide="play"></i><span>开始抢课</span>`;
  (document.getElementById("interval") as HTMLInputElement).disabled = state.running;
}

function renderSem() {
  const el = document.getElementById("semInfo")!;
  const s = state.semester;
  if (!s) {
    el.textContent = "";
    el.hidden = true;
    return;
  }
  el.hidden = false;
  const parts = [s.semester, s.courseType, s.stage];
  if (s.limit > 0) parts.push(`限选${s.limit}门`);
  parts.push(s.canDrop ? "可退选" : "不可退选");
  el.textContent = parts.filter(Boolean).join(" · ");
}

// ---------- 课程列表 ----------

function capacityCls(c: Course): string {
  const cap = c.pkrs || 0;
  const enr = c.jxbrs || 0;
  if (cap > 0 && enr >= cap) return "is-full";
  if (cap > 0 && cap - enr <= 5) return "is-warn";
  return "is-ok";
}

function visibleCourses(): Course[] {
  let list = state.courses;
  if (state.category) list = list.filter((c) => (c.kcflmc || c.kcdlmc) === state.category);
  if (state.search) {
    const q = state.search.toLowerCase();
    list = list.filter(
      (c) => c.kcmc.toLowerCase().includes(q) || (c.teaxm || "").toLowerCase().includes(q)
    );
  }
  return list;
}

function renderCourses() {
  const box = document.getElementById("courseList")!;
  if (state.loadingCourses) {
    box.innerHTML = `<div class="gh-empty"><span class="gh-spin"></span>正在加载课程…</div>`;
    return;
  }
  const list = visibleCourses();
  if (list.length === 0) {
    box.innerHTML = `<div class="gh-empty">${state.courses.length === 0 ? "暂无课程，点上方刷新试试" : "没有匹配的课程"}</div>`;
    return;
  }

  box.innerHTML = list
    .map((c) => {
      const cap = c.pkrs || 0;
      const enr = c.jxbrs || 0;
      const pct = cap > 0 ? Math.min(100, Math.round((enr / cap) * 100)) : 0;
      const inWish = state.wishlist.includes(c.kcrwdm);
      const grabbing = state.running && inWish;
      const cls = [capacityCls(c), grabbing ? "is-run" : ""].filter(Boolean).join(" ");
      const action = cap > 0 && enr >= cap
        ? `<button class="gh-btn gh-ghost" disabled>已满</button>`
        : inWish
          ? `<button class="gh-btn gh-ghost" data-act="unwish">移除</button>`
          : `<button class="gh-btn gh-primary" data-act="grab">抢</button>`;
      return `
        <div class="gh-row ${cls}" data-id="${esc(c.kcrwdm)}">
          <div class="gh-cell-main">
            <div class="gh-name">${grabbing ? `<span class="gh-run-dot"></span>` : ""}<button class="gh-name-btn" data-act="detail">${esc(c.kcmc)}</button></div>
            <div class="gh-meta">${esc(c.teaxm) || "未知教师"} · ${esc(c.xf) || "?"}学分 · ${esc(c.kcflmc || c.kcdlmc) || "-"}</div>
          </div>
          <div class="gh-cap">
            <div class="gh-cap-nums"><b>${enr}</b>/${cap}</div>
            <div class="gh-bar"><span style="width:${pct}%"></span></div>
          </div>
          <div class="gh-acts">
            <button class="gh-star ${inWish ? "on" : ""}" data-act="wish" aria-pressed="${inWish}" aria-label="收藏"><i data-lucide="star"></i></button>
            ${action}
          </div>
        </div>
      `;
    })
    .join("");

  box.querySelectorAll(".gh-row").forEach((row) => {
    const id = (row as HTMLElement).dataset.id!;
    row.addEventListener("click", (e) => {
      const act = (e.target as HTMLElement).closest<HTMLElement>("[data-act]")?.dataset.act;
      if (!act) return;
      e.stopPropagation();
      const c = courseById(id);
      if (!c) return;
      if (act === "wish") {
        if (!toggleWishlist(id)) toastErr(`已达到限选门数上限（${state.semester?.limit}门）`);
      } else if (act === "unwish") {
        toggleWishlist(id);
      } else if (act === "grab") {
        quickGrab(id);
      } else if (act === "detail") {
        openDetail(c);
      }
    });
  });
}

// ---------- 侧栏：收藏 + 已选 ----------

function bindSidebar(root: HTMLElement) {
  root.querySelector("#logoutBtn")!.addEventListener("click", () => App.logout());
  bindWishlistDnD();
}

function wishDot(c: Course | undefined): string {
  if (!c) return "";
  if (state.running) return "is-run";
  return capacityCls(c);
}

function renderWishlist() {
  const box = document.getElementById("wishList")!;
  document.getElementById("wishCount")!.textContent = String(state.wishlist.length);

  if (state.wishlist.length === 0) {
    box.innerHTML = `<div class="gh-empty-sm">在课程列表点 ★ 加入</div>`;
    return;
  }

  box.innerHTML = state.wishlist
    .map((id, i) => {
      const c = courseById(id);
      const name = c?.kcmc || id;
      return `
        <div class="gh-fav" draggable="true" data-id="${esc(id)}" data-index="${i}">
          <i data-lucide="grip-vertical" class="gh-grip"></i>
          <span class="gh-fav-txt">
            <span class="gh-fav-name">${esc(name)}</span>
            <span class="gh-fav-meta">${esc(c?.teaxm) || "-"} · ${esc(c?.xf) || "?"}学分</span>
          </span>
          <span class="gh-dot ${wishDot(c)}"></span>
          <button class="gh-x" data-act="remove" aria-label="移除"><i data-lucide="x"></i></button>
        </div>
      `;
    })
    .join("");

  box.querySelectorAll("[data-act='remove']").forEach((btn) => {
    btn.addEventListener("click", () => {
      toggleWishlist((btn.closest(".gh-fav") as HTMLElement).dataset.id!);
    });
  });
  bindWishlistDnD();
}

function bindWishlistDnD() {
  const box = document.getElementById("wishList")!;
  let dragIdx = -1;
  box.querySelectorAll<HTMLElement>(".gh-fav[draggable]").forEach((item) => {
    item.addEventListener("dragstart", () => {
      dragIdx = parseInt(item.dataset.index!, 10);
      item.classList.add("dragging");
    });
    item.addEventListener("dragend", () => {
      item.classList.remove("dragging");
      box.querySelectorAll(".drag-over").forEach((n) => n.classList.remove("drag-over"));
      dragIdx = -1;
    });
    item.addEventListener("dragover", (e) => {
      e.preventDefault();
      const idx = parseInt(item.dataset.index!, 10);
      if (idx !== dragIdx) item.classList.add("drag-over");
    });
    item.addEventListener("dragleave", () => item.classList.remove("drag-over"));
    item.addEventListener("drop", (e) => {
      e.preventDefault();
      item.classList.remove("drag-over");
      const to = parseInt(item.dataset.index!, 10);
      if (dragIdx !== -1 && dragIdx !== to) moveWishlist(dragIdx, to);
      dragIdx = -1;
    });
  });
}

function renderSelected() {
  const box = document.getElementById("selList")!;
  const limit = state.semester?.limit ?? 0;
  document.getElementById("selCount")!.textContent =
    limit > 0 ? `${state.selected.length}/${limit}` : String(state.selected.length);

  if (state.selected.length === 0) {
    box.innerHTML = `<div class="gh-empty-sm">暂无已选课程</div>`;
    return;
  }

  box.innerHTML = state.selected
    .map(
      (c) => `
      <div class="gh-fav gh-sel" data-id="${esc(c.kcrwdm)}">
        <span class="gh-fav-txt">
          <span class="gh-fav-name">${esc(c.kcmc)}</span>
          <span class="gh-fav-meta">${esc(c.teaxm) || "-"} · ${c.jxbrs || 0}/${c.pkrs || 0}</span>
        </span>
        <button class="gh-x gh-drop" data-act="drop" aria-label="退选"><span>退选</span></button>
      </div>
    `
    )
    .join("");

  box.querySelectorAll<HTMLElement>("[data-act='drop']").forEach((btn) => {
    btn.addEventListener("click", () => {
      const c = state.selected.find((x) => x.kcrwdm === (btn.closest(".gh-fav") as HTMLElement).dataset.id);
      if (!c) return;
      if (btn.dataset.confirm !== "1") {
        btn.dataset.confirm = "1";
        btn.classList.add("is-confirm");
        (btn.firstElementChild as HTMLElement).textContent = "确认?";
        setTimeout(() => {
          btn.dataset.confirm = "0";
          btn.classList.remove("is-confirm");
          (btn.firstElementChild as HTMLElement).textContent = "退选";
        }, 2600);
        return;
      }
      btn.classList.remove("is-confirm");
      (btn.firstElementChild as HTMLElement).textContent = "退选";
      btn.dataset.confirm = "0";
      dropCourse(c);
    });
  });
}

// ---------- 日志 ticker + 面板 ----------

function bindTicker(root: HTMLElement) {
  root.querySelector("#logToggle")!.addEventListener("click", () => {
    logOpen = !logOpen;
    renderLogPanel();
  });
}

function renderTicker() {
  const msg = document.getElementById("tickMsg")!;
  const dot = document.getElementById("tickDot")!;
  const count = document.getElementById("tickCount")!;

  const grabbed = state.logs.filter((l) => l.kind === "result" && l.success).length;
  document.querySelector(".gh-app")!.classList.toggle("is-running", state.running);

  const last = state.logs[state.logs.length - 1];
  if (state.running) {
    dot.className = "gh-tick-dot is-live";
    msg.textContent = last
      ? last.kind === "result"
        ? `${last.courseName || ""}：${last.message || ""}`
        : last.message || "抢课中"
      : `抢课中 · 清单 ${state.wishlist.length} 门`;
    count.textContent = `已抢到 ${grabbed} 门`;
  } else {
    dot.className = "gh-tick-dot";
    msg.textContent = last ? `${last.courseName || ""}：${last.message || ""}` : "就绪";
    count.textContent = state.logs.length > 0 ? `清单 ${state.wishlist.length} 门` : "";
  }
}

function renderLogPanel() {
  const panel = document.getElementById("logPanel")!;
  panel.classList.toggle("open", logOpen);
  if (!logOpen) return;

  const logs = [...state.logs].reverse();
  panel.innerHTML = `
    <div class="gh-log-head">
      <span>抢课日志</span>
      <span class="gh-badge">${state.logs.length}</span>
      <button id="logClear" class="gh-btn gh-ghost gh-xs">清空</button>
    </div>
    <div class="gh-log-list">
      ${
        logs.length === 0
          ? `<div class="gh-empty-sm">暂无日志</div>`
          : logs
              .map(
                (l) => `
        <div class="gh-log-entry ${l.kind === "result" ? (l.success ? "ok" : "fail") : "info"}">
          <span class="gh-log-time">${esc(l.time)}</span>
          <span class="gh-log-msg">${esc(l.courseName || "")}${l.kind === "result" ? "：" : " "}${esc(l.message || "")}</span>
        </div>
      `
              )
              .join("")
      }
    </div>
  `;
  panel.querySelector("#logClear")!.addEventListener("click", async () => {
    try { await App.clearGrabResults(); } catch { /* 忽略 */ }
    clearLogs();
  });
}

// ---------- 详情弹窗 ----------

let detailCourse: Course | null = null;

function bindModal(root: HTMLElement) {
  const modal = root.querySelector<HTMLElement>("#modal")!;
  modal.addEventListener("click", (e) => {
    if (e.target === modal) closeModal();
  });
}

function closeModal() {
  const modal = document.getElementById("modal")!;
  modal.classList.remove("open");
  detailCourse = null;
}

function openDetail(c: Course) {
  detailCourse = c;
  const modal = document.getElementById("modal")!;
  modal.innerHTML = `
    <div class="gh-modal-box">
      <div class="gh-modal-head">
        <h3>${esc(c.kcmc)}</h3>
        <button id="mClose" class="gh-icon-btn" aria-label="关闭"><i data-lucide="x"></i></button>
      </div>
      <div class="gh-modal-body">
        <div class="gh-detail-row"><span>教师</span><b>${esc(c.teaxm) || "-"}</b></div>
        <div class="gh-detail-row"><span>学分</span><b>${esc(c.xf) || "-"}</b></div>
        <div class="gh-detail-row"><span>分类</span><b>${esc(c.kcflmc || c.kcdlmc) || "-"}</b></div>
        <div class="gh-detail-row"><span>余量</span><b>${(c.pkrs || 0) - (c.jxbrs || 0)} / ${c.pkrs || 0}</b></div>
        <div class="gh-detail-row"><span>上课时间</span><div id="mSchedule" class="gh-schedule"><span class="gh-spin"></span></div></div>
      </div>
      <div class="gh-modal-foot">
        <button id="mAdd" class="gh-btn gh-primary">${state.wishlist.includes(c.kcrwdm) ? "移出愿望清单" : "加入愿望清单"}</button>
      </div>
    </div>
  `;
  modal.classList.add("open");
  refreshIcons();

  modal.querySelector("#mClose")!.addEventListener("click", closeModal);
  modal.querySelector("#mAdd")!.addEventListener("click", () => {
    if (!toggleWishlist(c.kcrwdm)) toastErr(`已达到限选门数上限（${state.semester?.limit}门）`);
    closeModal();
  });

  fetchSchedule(c.kcrwdm)
    .then((schedules) => {
      const box = modal.querySelector("#mSchedule")!;
      if (!schedules || schedules.length === 0) {
        box.textContent = "暂无上课时间信息";
        return;
      }
      const weeks = schedules.map((s) => parseInt(s.zc || "0", 10)).filter((w) => w > 0);
      const min = Math.min(...weeks), max = Math.max(...weeks);
      const weekRange = min === max ? `第${min}周` : `第${min}-${max}周`;
      const grouped = new Map<string, Set<string>>();
      schedules.forEach((s) => {
        const day = WEEK_DAYS[parseInt(s.xq || "0", 10) - 1] || `周${s.xq}`;
        if (!grouped.has(day)) grouped.set(day, new Set());
        (s.jcdm2 || "").split(",").forEach((j) => j && grouped.get(day)!.add(j));
      });
      box.innerHTML =
        `<div class="gh-weeks">${esc(weekRange)}</div>` +
        [...grouped.entries()]
          .map(([day, periods]) => `<span class="gh-schedule-item">${esc(day)} ${[...periods].sort().join(",")}节</span>`)
          .join("");
    })
    .catch(() => {
      const box = modal.querySelector("#mSchedule")!;
      box.textContent = "获取失败";
    });
}
