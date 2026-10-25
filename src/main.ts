// 入口：登录态路由 + mygo 事件订阅
import "./style.css";
import { App, events } from "./mygo";
import { renderLogin } from "./login";
import { renderMain } from "./app";
import { initToastHost, refreshIcons } from "./ui";
import {
  state, emit, resetRenders, loadAll, loadCourses, loadSelected,
  fetchGrabResults, pushLog, toastOk,
} from "./store";

const root = document.getElementById("app")!;
let view: "login" | "main" | null = null;

function showLogin(expired = false) {
  if (view === "login") return;
  view = "login";
  resetRenders();
  state.running = false;
  renderLogin(root, expired);
  refreshIcons();
}

async function showMain() {
  if (view === "main") return;
  view = "main";
  resetRenders();
  renderMain(root);
  await loadAll();
  await fetchGrabResults();
  emit();
}

initToastHost();

events.sessionChanged.on((ev) => {
  state.loggedIn = ev.loggedIn;
  if (ev.loggedIn) {
    showMain();
  } else {
    showLogin(ev.expired);
  }
});

events.grabUpdated.on((ev) => {
  pushLog(ev);
  if (ev.kind === "running") {
    state.running = true;
  } else if (ev.kind === "finished") {
    state.running = false;
    toastOk(ev.message || "抢课已结束");
    loadSelected();
    loadCourses();
  } else if (ev.kind === "result" && ev.success) {
    toastOk(`抢到「${ev.courseName}」`);
    loadSelected();
    loadCourses();
  }
  emit();
});

(async () => {
  try {
    const s = await App.status();
    state.version = s.version;
    if (s.loggedIn) await showMain();
    else showLogin();
  } catch {
    showLogin();
  }
})();
