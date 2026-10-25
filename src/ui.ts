// DOM 小工具 + toast + 图标刷新
import { createIcons, icons } from "lucide";

export function h(html: string): HTMLElement {
  const t = document.createElement("template");
  t.innerHTML = html.trim();
  return t.content.firstElementChild as HTMLElement;
}

export function esc(s: string | undefined | null): string {
  return (s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function refreshIcons() {
  createIcons({ icons, attrs: { width: 15, height: 15 } });
}

export function initToastHost() {
  document.addEventListener("gch-toast", (e) => {
    const { msg, kind } = (e as CustomEvent).detail as { msg: string; kind: "ok" | "err" };
    const t = h(`<div class="gh-toast ${kind === "err" ? "is-err" : ""}"></div>`);
    t.innerHTML = kind === "err" ? "✕" : "✓";
    const span = document.createElement("span");
    span.textContent = msg;
    t.appendChild(span);
    document.body.appendChild(t);
    setTimeout(() => t.classList.add("out"), 2600);
    setTimeout(() => t.remove(), 3000);
  });
}
