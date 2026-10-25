// 全局状态 + 动作：视图只读 state，动作改 state 后 emit。
import { App } from "./mygo";

export interface Course {
  kcrwdm: string;
  kcdm?: string;
  kcmc: string;
  teaxm?: string;
  kcdlmc?: string;
  kcflmc?: string;
  xf?: string;
  zxs?: string;
  pkrs?: number;
  jxbrs?: number;
  jxbdm?: string;
}

export interface CourseSchedule {
  kcrwdm: string;
  kcmc?: string;
  zc?: string;
  xq?: string;
  jcdm2?: string;
}

export interface GrabResult {
  course_id?: string;
  course_name?: string;
  success: boolean;
  message?: string;
  time?: string;
}

export interface SemesterInfo {
  semester: string;
  courseType: string;
  stage: string;
  limit: number;
  canDrop: boolean;
}

export interface GrabEvent {
  kind: string; // result | running | finished
  courseId?: string;
  courseName?: string;
  success: boolean;
  message?: string;
  time?: string;
}

export interface State {
  loggedIn: boolean;
  version: string;
  courses: Course[];
  wishlist: string[]; // 有序课程ID
  selected: Course[];
  semester: SemesterInfo | null;
  running: boolean;
  logs: GrabEvent[];
  category: string; // '' = 全部
  search: string;
  loadingCourses: boolean;
}

export const state: State = {
  loggedIn: false,
  version: "",
  courses: [],
  wishlist: [],
  selected: [],
  semester: null,
  running: false,
  logs: [],
  category: "",
  search: "",
  loadingCourses: false,
};

type Listener = () => void;
const listeners = new Set<Listener>();

export function onRender(fn: Listener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** 切换视图时清掉旧视图的渲染监听，避免渲染进已卸载的 DOM */
export function resetRenders() {
  listeners.clear();
}

export function emit() {
  listeners.forEach((fn) => fn());
}

// ---------- localStorage ----------

const LS_WISHLIST = "gch.wishlist";
const LS_INTERVAL = "gch.interval";

export function loadWishlist(ids: string[]) {
  try {
    const raw = localStorage.getItem(LS_WISHLIST);
    if (!raw) return;
    const saved = JSON.parse(raw) as string[];
    const known = new Set(ids);
    state.wishlist = saved.filter((id) => known.has(id));
  } catch {
    /* 忽略损坏的存档 */
  }
}

function saveWishlist() {
  localStorage.setItem(LS_WISHLIST, JSON.stringify(state.wishlist));
}

export function loadInterval(): number {
  const v = parseInt(localStorage.getItem(LS_INTERVAL) || "1", 10);
  return v >= 1 && v <= 60 ? v : 1;
}

export function saveInterval(sec: number) {
  localStorage.setItem(LS_INTERVAL, String(sec));
}

// ---------- 动作 ----------

export function courseById(id: string): Course | undefined {
  return state.courses.find((c) => c.kcrwdm === id);
}

export function toggleWishlist(id: string): boolean {
  const i = state.wishlist.indexOf(id);
  if (i >= 0) {
    state.wishlist.splice(i, 1);
  } else {
    const limit = state.semester?.limit ?? 0;
    if (limit > 0 && state.selected.length + state.wishlist.length >= limit) {
      return false; // 超限，由调用方提示
    }
    state.wishlist.push(id);
  }
  saveWishlist();
  emit();
  return true;
}

export function moveWishlist(from: number, to: number) {
  if (to < 0 || to >= state.wishlist.length || from === to) return;
  const [item] = state.wishlist.splice(from, 1);
  state.wishlist.splice(to, 0, item);
  saveWishlist();
  emit();
}

export function clearWishlist() {
  state.wishlist = [];
  saveWishlist();
  emit();
}

export function setCategory(cat: string) {
  state.category = cat;
  emit();
}

export function setSearch(q: string) {
  state.search = q;
  emit();
}

export function pushLog(ev: GrabEvent) {
  state.logs.push(ev);
  if (state.logs.length > 300) state.logs.splice(0, state.logs.length - 300);
}

export function clearLogs() {
  state.logs = [];
  emit();
}

// ---------- IPC 封装 ----------

export async function importSession(jsessionid: string): Promise<void> {
  await App.importSession(jsessionid); // 成功后 sessionChanged 事件驱动切换视图
}

export async function loadAll() {
  await Promise.all([loadSemester(), loadCourses(), loadSelected()]);
  loadWishlist(state.courses.map((c) => c.kcrwdm));
  emit();
}

export async function loadSemester() {
  try {
    state.semester = await App.semesterInfo();
  } catch (e) {
    state.semester = null;
  }
}

export async function loadCourses() {
  state.loadingCourses = true;
  emit();
  try {
    state.courses = await App.listCourses();
  } catch (e) {
    toastErr(errMsg(e));
  } finally {
    state.loadingCourses = false;
    emit();
  }
}

export async function loadSelected() {
  try {
    state.selected = await App.selectedCourses();
  } catch (e) {
    /* 静默，登录态失效会走 sessionChanged */
  }
}

export async function startGrab() {
  if (state.wishlist.length === 0) {
    toastErr("先在课程列表点 ★ 或「抢」加入清单");
    return;
  }
  const limit = state.semester?.limit ?? 0;
  if (limit > 0 && state.selected.length >= limit) {
    toastErr(`已选课程已达上限（${limit}门）`);
    return;
  }
  const names: Record<string, string> = {};
  state.wishlist.forEach((id) => {
    const c = courseById(id);
    if (c) names[id] = c.kcmc;
  });
  try {
    await App.startGrab([...state.wishlist], names, loadInterval());
    state.running = true;
    emit();
  } catch (e) {
    toastErr(errMsg(e));
  }
}

export async function stopGrab() {
  try {
    await App.stopGrab();
  } catch {
    /* finished 事件会兜底 */
  }
  state.running = false;
  emit();
}

/** 行内「抢」：加入清单；未在抢课时直接开抢 */
export async function quickGrab(id: string) {
  const wasIn = state.wishlist.includes(id);
  if (!wasIn) {
    const ok = toggleWishlist(id);
    if (!ok) {
      toastErr(`已达到限选门数上限（${state.semester?.limit}门）`);
      return;
    }
  }
  if (state.running) {
    if (!wasIn) toastOk("已加入清单，本次抢课结束后自动纳入下一轮");
    return;
  }
  await startGrab();
}

export async function dropCourse(c: Course) {
  const r = await App.dropCourse(c.kcrwdm, c.kcmc, c.jxbdm || "");
  if (r.success) {
    toastOk(r.message || "退选成功");
    await loadSelected();
  } else {
    toastErr(r.message || "退选失败");
  }
}

export async function fetchSchedule(kcrwdm: string): Promise<CourseSchedule[]> {
  return App.courseSchedule(kcrwdm);
}

export async function fetchGrabResults() {
  try {
    state.running = await App.grabRunning();
    const results = await App.grabResults();
    state.logs = results.map((r) => ({
      kind: "result",
      courseId: r.course_id,
      courseName: r.course_name,
      success: r.success,
      message: r.message,
      time: r.time ? r.time.slice(11, 19) : "",
    }));
  } catch {
    /* 未登录时静默 */
  }
}

// ---------- toast / 错误 ----------

export function errMsg(e: unknown): string {
  const anyE = e as { message?: string };
  return anyE?.message || String(e);
}

export function toastOk(msg: string) {
  showToast(msg, "ok");
}

export function toastErr(msg: string) {
  showToast(msg, "err");
}

function showToast(msg: string, kind: "ok" | "err") {
  document.dispatchEvent(new CustomEvent("gch-toast", { detail: { msg, kind } }));
}
