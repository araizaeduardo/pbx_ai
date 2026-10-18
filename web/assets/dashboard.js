// PBX AI Dashboard Client-side Controller

let sessionsData = [];
let autoRefreshTimer = null;

document.addEventListener("DOMContentLoaded", () => {
  initTabs();
  initDrawer();
  initHelpModal();
  initFilters();

  loadAllData();
  // Auto-refresh every 5 seconds
  autoRefreshTimer = setInterval(loadAllData, 5000);

  document.getElementById("btn-refresh").addEventListener("click", () => {
    loadAllData();
  });
});

function initTabs() {
  const tabs = document.querySelectorAll(".tab-btn");
  tabs.forEach(tab => {
    tab.addEventListener("click", () => {
      tabs.forEach(t => t.classList.remove("active"));
      tab.classList.add("active");

      document.querySelectorAll(".tab-content").forEach(content => {
        content.style.display = "none";
      });

      const targetId = tab.getAttribute("data-tab");
      const targetContent = document.getElementById(targetId);
      if (targetContent) {
        targetContent.style.display = "block";
      }

      if (targetId === "tab-logs") {
        fetchLogs();
      } else if (targetId === "tab-sessions") {
        fetchSessions();
      }
    });
  });
}

function initDrawer() {
  const backdrop = document.getElementById("session-drawer-backdrop");
  const closeBtn = document.getElementById("drawer-close-btn");

  closeBtn.addEventListener("click", () => {
    backdrop.classList.remove("active");
  });

  backdrop.addEventListener("click", (e) => {
    if (e.target === backdrop) {
      backdrop.classList.remove("active");
    }
  });
}

function initFilters() {
  document.getElementById("btn-apply-filter").addEventListener("click", () => {
    fetchLogs();
  });

  document.getElementById("log-channel-filter").addEventListener("change", () => {
    fetchLogs();
  });

  document.getElementById("log-search-input").addEventListener("keyup", (e) => {
    if (e.key === "Enter") {
      fetchLogs();
    }
  });
}

async function loadAllData() {
  await fetchStats();
  await fetchSessions();
  // If logs tab is currently active, refresh logs too
  const activeTab = document.querySelector(".tab-btn.active");
  if (activeTab && activeTab.getAttribute("data-tab") === "tab-logs") {
    await fetchLogs();
  }
}

async function fetchStats() {
  try {
    const res = await fetch("/api/dashboard/stats");
    if (!res.ok) throw new Error("Stats request failed");
    const data = await res.json();

    // KPIs
    const reqs = data.metrics.total_requests || data.metrics.rt_requests || 0;
    document.getElementById("kpi-requests").textContent = reqs.toLocaleString();

    const tokens = data.metrics.total_tokens || 0;
    document.getElementById("kpi-tokens").textContent = tokens.toLocaleString();

    const cost = data.metrics.estimated_cost_usd || 0;
    document.getElementById("kpi-cost").textContent = `Est. cost: $${cost.toFixed(4)}`;

    const lat = data.metrics.avg_latency_ms || 0;
    document.getElementById("kpi-latency").textContent = `${lat} ms`;

    const active = data.system.active_sessions || 0;
    document.getElementById("kpi-sessions").textContent = active.toLocaleString();

    const uptimeSec = data.system.uptime_seconds || 0;
    document.getElementById("kpi-uptime").textContent = `Uptime: ${formatUptime(uptimeSec)}`;

    // Channels progress bars
    const counts = data.metrics.channel_counts || {};
    const webCnt = (counts["web"] || 0) + (counts["chat"] || 0);
    const astCnt = (counts["asterisk"] || 0) + (counts["pbx"] || 0);
    const metaCnt = (counts["meta"] || 0) + (counts["whatsapp"] || 0);
    const tgCnt = counts["telegram"] || 0;
    const maxVal = Math.max(webCnt, astCnt, metaCnt, tgCnt, 1);

    document.getElementById("ch-web-count").textContent = `${webCnt} msgs`;
    document.getElementById("ch-web-fill").style.width = `${Math.min(100, Math.round((webCnt / maxVal) * 100))}%`;

    document.getElementById("ch-asterisk-count").textContent = `${astCnt} calls`;
    document.getElementById("ch-asterisk-fill").style.width = `${Math.min(100, Math.round((astCnt / maxVal) * 100))}%`;

    document.getElementById("ch-meta-count").textContent = `${metaCnt} msgs`;
    document.getElementById("ch-meta-fill").style.width = `${Math.min(100, Math.round((metaCnt / maxVal) * 100))}%`;

    document.getElementById("ch-tg-count").textContent = `${tgCnt} msgs`;
    document.getElementById("ch-tg-fill").style.width = `${Math.min(100, Math.round((tgCnt / maxVal) * 100))}%`;

    // Providers status
    updateProviderBadge("prov-llm-badge", data.providers.llm.configured, data.providers.llm.model || "Configured");
    updateProviderBadge("prov-asterisk-badge", data.providers.asterisk.enabled, data.providers.asterisk.enabled ? "Active" : "Disabled");
    updateProviderBadge("prov-meta-badge", data.providers.meta.configured, data.providers.meta.configured ? "Connected" : "Not Set");
    updateProviderBadge("prov-tg-badge", data.providers.telegram.configured, data.providers.telegram.configured ? "Connected" : "Not Set");
    updateProviderBadge("prov-db-badge", true, `${data.providers.database.type} (${data.providers.database.name})`);

    // Diagnostics
    document.getElementById("diag-alloc").textContent = `${data.system.alloc_mb.toFixed(2)} MB`;
    document.getElementById("diag-sys").textContent = `${data.system.sys_mb.toFixed(2)} MB`;
    document.getElementById("diag-goroutines").textContent = data.system.goroutines;
    document.getElementById("diag-database").textContent = `${data.providers.database.type} (${data.providers.database.name})`;

  } catch (err) {
    console.error("fetchStats error:", err);
    document.getElementById("system-status-text").textContent = "Service Degraded";
    document.getElementById("system-status-pill").style.borderColor = "rgba(244, 63, 94, 0.4)";
  }
}

function updateProviderBadge(elId, isOk, label) {
  const el = document.getElementById(elId);
  if (!el) return;
  el.textContent = label;
  el.className = "badge " + (isOk ? "badge-success" : "badge-error");
}

async function fetchSessions() {
  try {
    const res = await fetch("/api/dashboard/sessions");
    if (!res.ok) return;
    sessionsData = await res.json();

    const tbody = document.getElementById("sessions-table-body");
    document.getElementById("sessions-count-label").textContent = `${sessionsData.length} active sessions`;

    if (!sessionsData || sessionsData.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align: center; color: var(--text-dim); padding: 30px;">No active in-memory sessions right now.</td></tr>`;
      return;
    }

    tbody.innerHTML = sessionsData.map((s, idx) => `
      <tr>
        <td><code>${escapeHtml(s.user_id)}</code></td>
        <td><span class="badge badge-chat">${s.message_count} msgs</span></td>
        <td style="color: var(--text-muted); font-size: 0.8rem;">${escapeHtml(s.last_active || "—")}</td>
        <td style="max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
          ${escapeHtml(s.last_message || "—")}
        </td>
        <td>
          <button class="btn btn-inspect" onclick="openSessionDrawer(${idx})" style="padding: 4px 10px; font-size: 0.75rem;">
            Inspect
          </button>
        </td>
      </tr>
    `).join("");
  } catch (err) {
    console.error("fetchSessions error:", err);
  }
}

function openSessionDrawer(idx) {
  const session = sessionsData[idx];
  if (!session) return;

  document.getElementById("drawer-user-id").textContent = `Session: ${session.user_id}`;
  document.getElementById("drawer-user-sub").textContent = `${session.message_count} messages | Last: ${session.last_active || "—"}`;

  const container = document.getElementById("drawer-bubbles");
  if (!session.history || session.history.length === 0) {
    container.innerHTML = `<div style="text-align: center; color: var(--text-dim); margin-top: 40px;">No messages stored.</div>`;
  } else {
    container.innerHTML = session.history.map(m => `
      <div class="chat-bubble ${m.role === 'user' ? 'bubble-user' : 'bubble-assistant'}">
        <div style="font-size: 0.7rem; opacity: 0.7; margin-bottom: 4px; font-weight: 700;">
          ${escapeHtml(m.role.toUpperCase())}
        </div>
        <div>${escapeHtml(m.content || '')}</div>
      </div>
    `).join("");
  }

  document.getElementById("session-drawer-backdrop").classList.add("active");
}

async function fetchLogs() {
  try {
    const channel = document.getElementById("log-channel-filter").value;
    const search = document.getElementById("log-search-input").value;
    const url = `/api/dashboard/logs?channel=${encodeURIComponent(channel)}&search=${encodeURIComponent(search)}`;

    const res = await fetch(url);
    if (!res.ok) return;
    const logs = await res.json();

    const tbody = document.getElementById("logs-table-body");
    if (!logs || logs.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align: center; color: var(--text-dim); padding: 30px;">No matching logs found in aiLogs table.</td></tr>`;
      return;
    }

    tbody.innerHTML = logs.map(l => {
      const ch = l.channel || "chat";
      const badgeClass = "badge-" + ch;
      const isErr = l.error && l.error !== "";
      return `
        <tr>
          <td><span style="color: var(--text-dim);">#${l.ejaId}</span></td>
          <td style="color: var(--text-muted); font-size: 0.8rem; white-space: nowrap;">${escapeHtml(l.ejaLog || "")}</td>
          <td><span class="badge ${badgeClass}">${escapeHtml(ch)}</span></td>
          <td><code>${escapeHtml(l.source || "")}</code></td>
          <td style="color: var(--text-muted); font-size: 0.8rem;">${escapeHtml(l.model || "—")}</td>
          <td>${l.latency_ms > 0 ? l.latency_ms + ' ms' : '—'}</td>
          <td>${l.tokens > 0 ? l.tokens : '—'}</td>
          <td style="max-width: 380px;">
            <div style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
              ${escapeHtml(l.message || "")}
            </div>
            ${isErr ? `<div class="badge badge-error" style="margin-top: 4px; display: inline-block;">${escapeHtml(l.error)}</div>` : ''}
          </td>
        </tr>
      `;
    }).join("");
  } catch (err) {
    console.error("fetchLogs error:", err);
  }
}

function formatUptime(seconds) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

function escapeHtml(str) {
  if (!str) return "";
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function initHelpModal() {
  const helpBtn = document.getElementById("btn-help");
  const backdrop = document.getElementById("help-modal-backdrop");
  const closeBtn = document.getElementById("help-close-btn");
  const searchInput = document.getElementById("help-search-input");
  const tabs = document.querySelectorAll(".help-tab-btn");
  const sections = document.querySelectorAll(".help-section");

  if (!helpBtn || !backdrop) return;

  helpBtn.addEventListener("click", () => {
    backdrop.classList.add("active");
    if (searchInput) searchInput.focus();
  });

  if (closeBtn) {
    closeBtn.addEventListener("click", () => {
      backdrop.classList.remove("active");
    });
  }

  backdrop.addEventListener("click", (e) => {
    if (e.target === backdrop) {
      backdrop.classList.remove("active");
    }
  });

  window.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && backdrop.classList.contains("active")) {
      backdrop.classList.remove("active");
    }
  });

  // Help tab navigation
  tabs.forEach(tab => {
    tab.addEventListener("click", () => {
      tabs.forEach(t => t.classList.remove("active"));
      tab.classList.add("active");

      const targetId = tab.getAttribute("data-help-tab");
      sections.forEach(sec => {
        sec.classList.remove("active");
        if (sec.id === targetId) {
          sec.classList.add("active");
        }
      });

      // Scroll body back to top on tab change
      const helpBody = document.getElementById("help-body");
      if (helpBody) helpBody.scrollTop = 0;
    });
  });

  // Live search in help modal
  if (searchInput) {
    searchInput.addEventListener("input", (e) => {
      const q = e.target.value.toLowerCase().trim();
      if (!q) {
        // Restore current tab
        const activeTab = document.querySelector(".help-tab-btn.active");
        const targetId = activeTab ? activeTab.getAttribute("data-help-tab") : "help-intro";
        sections.forEach(sec => {
          sec.classList.toggle("active", sec.id === targetId);
          sec.querySelectorAll(".help-card, .code-container, tr").forEach(el => el.style.display = "");
        });
        return;
      }

      // In search mode, show all sections that have matching text
      sections.forEach(sec => {
        let hasMatch = false;
        const elements = sec.querySelectorAll(".help-card, .code-container, tr, p");
        elements.forEach(el => {
          const match = el.textContent.toLowerCase().includes(q);
          el.style.display = match ? "" : "none";
          if (match) hasMatch = true;
        });

        sec.classList.toggle("active", hasMatch);
      });
    });
  }
}

// Global copy-to-clipboard for code blocks
window.copyCode = function(button) {
  const container = button.closest(".code-container");
  if (!container) return;
  const codeEl = container.querySelector("pre code");
  if (!codeEl) return;

  const text = codeEl.innerText;
  navigator.clipboard.writeText(text).then(() => {
    const originalText = button.textContent;
    button.textContent = "¡Copiado! ✓";
    button.style.background = "#10b981";
    button.style.color = "#ffffff";
    setTimeout(() => {
      button.textContent = originalText;
      button.style.background = "";
      button.style.color = "";
    }, 2000);
  }).catch(() => {
    // Fallback
    const ta = document.createElement("textarea");
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
    button.textContent = "¡Copiado! ✓";
    setTimeout(() => { button.textContent = "Copiar"; }, 2000);
  });
};
