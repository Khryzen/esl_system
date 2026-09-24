(() => {
  "use strict";

  const START_HOUR = 0;
  const END_HOUR = 24;
  const ROW_HEIGHT = 64;

  const scheduleUrl = window.location.pathname;

  const loadingEl = document.getElementById("calendarLoading");
  const scrollEl = document.getElementById("calendarScroll");
  const monthEl = document.getElementById("calendarMonth");
  const monthGrid = document.getElementById("monthGrid");
  const labelsEl = document.getElementById("calendarLabels");
  const cellsEl = document.getElementById("calendarCells");
  const dateLabelEl = document.getElementById("calendarDateLabel");
  const dateInput = document.getElementById("calendarDateInput");
  const hintEl = document.getElementById("calendarHint");

  const viewDayBtn = document.getElementById("viewDayBtn");
  const viewMonthBtn = document.getElementById("viewMonthBtn");

  const modal = document.getElementById("classModal");
  const form = document.getElementById("classForm");
  const submitBtn = document.getElementById("classSubmitBtn");
  const enrollmentSelect = document.getElementById("classEnrollment");
  const enrollmentEmpty = document.getElementById("classEnrollmentEmpty");
  const classDateInput = document.getElementById("classDate");
  const classStartInput = document.getElementById("classStartTime");
  const durationEl = document.getElementById("classDuration");
  const endTimeEl = document.getElementById("classEndTime");
  const remainingEl = document.getElementById("classRemaining");

  const detailModal = document.getElementById("classDetailModal");
  const detailBody = document.getElementById("classDetailBody");

  if (!scrollEl || !cellsEl) return;

  const state = {
    view: "day",
    date: startOfDay(new Date()),
    classes: [],
    enrollments: [],
    counts: {},
  };

  let detailClassId = 0;

  function startOfDay(d) {
    return new Date(d.getFullYear(), d.getMonth(), d.getDate());
  }
  function pad2(n) {
    return String(n).padStart(2, "0");
  }
  function toDateStr(d) {
    return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
  }
  function parseLocalDate(str) {
    const [y, m, d] = str.split("-").map(Number);
    return new Date(y, m - 1, d);
  }
  function parseServerTime(s) {
    return new Date(s);
  }
  function formatHour(hour) {
    const p = hour < 12 ? "AM" : "PM";
    const h = hour % 12 === 0 ? 12 : hour % 12;
    return `${h}:00 ${p}`;
  }
  function formatClock(d) {
    const p = d.getHours() < 12 ? "AM" : "PM";
    const h = d.getHours() % 12 === 0 ? 12 : d.getHours() % 12;
    return `${h}:${pad2(d.getMinutes())} ${p}`;
  }
  function addMinutes(hhmm, minutes) {
    const [h, m] = hhmm.split(":").map(Number);
    return new Date(2000, 0, 1, h, m + minutes);
  }
  function escapeHtml(s) {
    const div = document.createElement("div");
    div.textContent = s ?? "";
    return div.innerHTML;
  }

  function classColors(status) {
    if (status === "present")
      return "border-green-500 bg-green-50 text-green-900 dark:border-green-400 dark:bg-green-950/50 dark:text-green-100";
    if (status === "absent")
      return "border-red-500 bg-red-50 text-red-900 dark:border-red-400 dark:bg-red-950/50 dark:text-red-100";
    return "border-blue-500 bg-blue-50 text-blue-900 dark:border-blue-400 dark:bg-blue-950/50 dark:text-blue-100";
  }

  // TinyMCE for the four feedback fields (Grammar Corrections, Recommendation, Homework,
  // Remarks). Scoped to #classDetailBody so it never touches anything outside the detail
  // modal. Guarded with typeof checks throughout: if the CDN script in dashboard.html
  // didn't load (offline, blocked, etc.), these quietly no-op and the fields stay plain
  // textareas rather than throwing and breaking the rest of the modal.
  const TINYMCE_SELECTOR = "#classDetailBody textarea.tinymce-field";

  function initFeedbackEditors() {
    if (typeof tinymce === "undefined") return;
    tinymce.init({
      selector: TINYMCE_SELECTOR,
      height: 180,
      menubar: false,
      statusbar: false,
      branding: false,
      plugins: "lists",
      toolbar: "bold italic underline | bullist numlist | removeformat",
    });
  }

  function destroyFeedbackEditors() {
    if (typeof tinymce === "undefined") return;
    tinymce.remove(TINYMCE_SELECTOR);
  }

  let gridBuilt = false;
  function buildGridSkeleton() {
    if (gridBuilt) return;
    gridBuilt = true;

    const hourCount = END_HOUR - START_HOUR;
    cellsEl.style.height = `${hourCount * ROW_HEIGHT}px`;

    for (let hour = START_HOUR; hour < END_HOUR; hour++) {
      const label = document.createElement("div");
      label.style.height = `${ROW_HEIGHT}px`;
      label.className =
        "flex items-start justify-end pr-2 pt-0 -translate-y-2.5 text-xs text-gray-400 dark:text-gray-500";
      label.textContent = formatHour(hour);
      labelsEl.appendChild(label);

      const cell = document.createElement("div");
      cell.style.height = `${ROW_HEIGHT}px`;
      cell.dataset.hour = String(hour);
      cell.setAttribute("role", "button");
      cell.setAttribute("tabindex", "0");
      cell.setAttribute(
        "aria-label",
        `Schedule a class at ${formatHour(hour)}`,
      );
      cell.className =
        "cursor-pointer border-b border-gray-100 transition hover:bg-blue-50/60 dark:border-gray-800 dark:hover:bg-blue-950/20";
      cellsEl.appendChild(cell);
    }
  }

  function clearEvents() {
    cellsEl.querySelectorAll(".calendar-event").forEach((el) => el.remove());
  }

  function renderEvents() {
    clearEvents();

    const gridStart = new Date(state.date);
    gridStart.setHours(START_HOUR, 0, 0, 0);
    const gridEnd = new Date(state.date);
    gridEnd.setHours(END_HOUR, 0, 0, 0);
    const gridHeight = (END_HOUR - START_HOUR) * ROW_HEIGHT;
    const pxPerMinute = ROW_HEIGHT / 60;

    for (const cls of state.classes) {
      const start = parseServerTime(cls.start);
      const end = parseServerTime(cls.end);
      if (end <= gridStart || start >= gridEnd) continue;

      const top = Math.max(0, (start - gridStart) / 60000) * pxPerMinute;
      const bottom = Math.min(
        gridHeight,
        ((Math.max(end, start) - gridStart) / 60000) * pxPerMinute,
      );

      const block = document.createElement("div");
      block.className = `calendar-event cursor-pointer absolute inset-x-2 overflow-hidden rounded-lg border-l-4 px-2 py-1 text-xs leading-tight shadow-sm transition hover:brightness-95 ${classColors(cls.status)}`;
      block.style.top = `${top}px`;
      block.style.height = `${Math.max(bottom - top, 18)}px`;
      block.dataset.classId = String(cls.id);
      block.title = `${cls.student} — ${cls.course}\n${formatClock(start)}–${formatClock(end)}`;
      block.innerHTML = `
        <p class="truncate font-medium">${escapeHtml(cls.student)}</p>
        <p class="truncate opacity-80">${escapeHtml(cls.course)}</p>
      `;
      cellsEl.appendChild(block);
    }
  }

  function monthGridStart(d) {
    const first = new Date(d.getFullYear(), d.getMonth(), 1);
    const start = new Date(first);
    start.setDate(first.getDate() - first.getDay());
    return startOfDay(start);
  }

  function renderMonth() {
    monthGrid.innerHTML = "";

    for (const wd of ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]) {
      const el = document.createElement("div");
      el.className =
        "bg-gray-50 py-2 text-center text-xs font-medium text-gray-500 dark:bg-gray-800/60 dark:text-gray-400";
      el.textContent = wd;
      monthGrid.appendChild(el);
    }

    const start = monthGridStart(state.date);
    const today = startOfDay(new Date());

    for (let i = 0; i < 42; i++) {
      const d = new Date(start);
      d.setDate(start.getDate() + i);
      const key = toDateStr(d);
      const count = state.counts[key] || 0;

      const cell = document.createElement("button");
      cell.type = "button";
      cell.dataset.date = key;
      cell.className =
        "flex min-h-[88px] flex-col gap-1 bg-white p-2 text-left transition hover:bg-blue-50 dark:bg-gray-900 dark:hover:bg-blue-950/20";

      const inMonth = d.getMonth() === state.date.getMonth();
      const isToday = d.getTime() === today.getTime();

      const num = document.createElement("span");
      num.className = isToday
        ? "flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white"
        : "text-sm " +
          (inMonth
            ? "text-gray-900 dark:text-white"
            : "text-gray-400 dark:text-gray-600");
      num.textContent = String(d.getDate());
      cell.appendChild(num);

      if (count > 0) {
        const badge = document.createElement("span");
        badge.className =
          "self-start rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-medium text-blue-700 dark:bg-blue-950/60 dark:text-blue-200";
        badge.textContent = `${count} class${count === 1 ? "" : "es"}`;
        cell.appendChild(badge);
      }

      monthGrid.appendChild(cell);
    }
  }

  async function loadSchedule() {
    loadingEl.textContent = "Loading schedule…";
    loadingEl.classList.remove("hidden");
    scrollEl.classList.add("hidden");
    monthEl.classList.add("hidden");

    try {
      const url =
        state.view === "month"
          ? `${scheduleUrl}?start=${toDateStr(monthGridStart(state.date))}&days=42&view=month`
          : `${scheduleUrl}?start=${toDateStr(state.date)}&days=1`;

      const response = await fetch(url, { method: "POST" });
      const contentType = response.headers.get("content-type") || "";
      if (!contentType.includes("application/json"))
        throw new Error(`Server returned ${response.status}.`);

      const data = await response.json();
      if (data.status !== "ok")
        throw new Error(data.message || "Could not load the schedule.");

      if (state.view === "month") {
        state.counts = data.counts || {};
        renderMonth();
        monthEl.classList.remove("hidden");
      } else {
        state.classes = data.classes || [];
        state.enrollments = data.enrollments || [];
        renderEvents();
        scrollEl.classList.remove("hidden");
        scrollEl.scrollTop = 7 * ROW_HEIGHT;
      }

      loadingEl.classList.add("hidden");
    } catch (error) {
      console.error("Error loading schedule:", error);
      loadingEl.textContent = "Could not load the schedule. Try again.";
    }
  }

  function applyViewButtonStyles() {
    const active = "bg-blue-600 text-white shadow-sm";
    const inactive =
      "text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800";
    viewDayBtn.className = `rounded-md px-3 py-1.5 text-sm font-medium transition ${state.view === "day" ? active : inactive}`;
    viewMonthBtn.className = `rounded-md px-3 py-1.5 text-sm font-medium transition ${state.view === "month" ? active : inactive}`;
    hintEl.classList.toggle("hidden", state.view === "month");
  }

  function setView(view) {
    if (state.view === view) return;
    state.view = view;
    applyViewButtonStyles();
    refreshDateControls();
    loadSchedule();
  }

  function refreshDateControls() {
    dateInput.value = toDateStr(state.date);
    dateLabelEl.textContent =
      state.view === "month"
        ? state.date.toLocaleDateString(undefined, {
            month: "long",
            year: "numeric",
          })
        : state.date.toLocaleDateString(undefined, {
            weekday: "long",
            year: "numeric",
            month: "long",
            day: "numeric",
          });
  }

  function goToDate(date) {
    state.date = startOfDay(date);
    refreshDateControls();
    loadSchedule();
  }

  function shiftDate(delta) {
    const d = new Date(state.date);
    if (state.view === "month") d.setMonth(d.getMonth() + delta);
    else d.setDate(d.getDate() + delta);
    goToDate(d);
  }

  function populateEnrollmentSelect() {
    enrollmentSelect.innerHTML =
      '<option value="" disabled selected>-- Select an enrollment --</option>';
    const has = state.enrollments.length > 0;
    enrollmentEmpty.classList.toggle("hidden", has);
    enrollmentSelect.disabled = !has;
    submitBtn.disabled = !has;

    for (const en of state.enrollments) {
      const option = document.createElement("option");
      option.value = en.id;
      option.textContent = `${en.student} — ${en.course} (${en.package}), ${en.classes_remaining} left`;
      option.dataset.durationMinutes = en.duration_minutes;
      option.dataset.classesRemaining = en.classes_remaining;
      enrollmentSelect.appendChild(option);
    }
  }

  function updateClassPreview() {
    const option = enrollmentSelect.selectedOptions[0];
    const duration = option ? Number(option.dataset.durationMinutes) : 0;

    if (!option || !duration || !classStartInput.value) {
      durationEl.textContent = "—";
      endTimeEl.textContent = "—";
      remainingEl.textContent = "";
      return;
    }
    durationEl.textContent = `${duration} min`;
    endTimeEl.textContent = formatClock(
      addMinutes(classStartInput.value, duration),
    );
    const remaining = Number(option.dataset.classesRemaining);
    remainingEl.textContent = `${remaining} class${remaining === 1 ? "" : "es"} remaining on this enrollment`;
  }

  function openAddModal(hour) {
    form.reset();
    populateEnrollmentSelect();
    classDateInput.value = toDateStr(state.date);
    classStartInput.value = `${pad2(hour)}:00`;
    updateClassPreview();
    modal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
    enrollmentSelect.focus();
  }

  function closeModal() {
    modal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  function renderDetailBody(cls) {
    // Detaches the previous class's editors before their textareas are replaced below —
    // otherwise TinyMCE keeps a reference to DOM nodes that no longer exist.
    destroyFeedbackEditors();

    const start = parseServerTime(cls.start);
    const end = parseServerTime(cls.end);

    const infoRows = [
      ["Student", cls.student || "—"],
      ["Course", cls.course || "—"],
      ["Package", cls.package || "—"],
      ["Reference", cls.reference || "—"],
      [
        "Date",
        start.toLocaleDateString(undefined, {
          weekday: "long",
          year: "numeric",
          month: "long",
          day: "numeric",
        }),
      ],
      ["Time", `${formatClock(start)} – ${formatClock(end)}`],
      ["Duration", `${cls.duration_minutes} min`],
    ];

    const infoHtml = `
      <div class="space-y-3">
        ${infoRows
          .map(
            ([l, v]) => `
          <div class="flex justify-between gap-4 border-b border-gray-100 pb-2 last:border-0 dark:border-gray-800">
            <span class="text-gray-500 dark:text-gray-400">${escapeHtml(l)}</span>
            <span class="text-right font-medium text-gray-900 dark:text-white">${escapeHtml(String(v))}</span>
          </div>`,
          )
          .join("")}
      </div>`;

    let attendanceHtml = "";
    if (!cls.status) {
      attendanceHtml = `
        <div class="rounded-xl border border-gray-200 p-4 dark:border-gray-800">
          <p class="mb-3 text-sm font-medium text-gray-900 dark:text-white">Attendance</p>
          <div class="flex gap-2">
            <button type="button" data-attendance="present" data-class-id="${cls.id}"
              class="flex-1 rounded-xl bg-green-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-green-700">
              Mark Present
            </button>
            <button type="button" data-attendance="absent" data-class-id="${cls.id}"
              class="flex-1 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-red-700">
              Mark Absent
            </button>
          </div>
        </div>`;
    } else if (cls.status === "present") {
      attendanceHtml = `
        <div class="flex items-center justify-between rounded-xl border border-green-200 bg-green-50 px-4 py-3 dark:border-green-900/60 dark:bg-green-950/40">
          <span class="text-sm font-medium text-green-900 dark:text-green-200">Marked Present</span>
          <span class="rounded-full bg-green-600 px-2 py-0.5 text-xs font-semibold text-white">Present</span>
        </div>`;
    } else if (cls.status === "absent") {
      attendanceHtml = `
        <div class="flex items-center justify-between rounded-xl border border-red-200 bg-red-50 px-4 py-3 dark:border-red-900/60 dark:bg-red-950/40">
          <span class="text-sm font-medium text-red-900 dark:text-red-200">
            Marked Absent ${cls.credit_refunded ? "· class credit refunded" : "· no refund"}
          </span>
          <span class="rounded-full bg-red-600 px-2 py-0.5 text-xs font-semibold text-white">Absent</span>
        </div>`;
    }

    let feedbackHtml = "";
    if (cls.status === "present") {
      const a = cls.assessment || {};
      feedbackHtml = `
        <div class="rounded-xl border border-gray-200 p-4 dark:border-gray-800">
          <p class="mb-3 text-sm font-medium text-gray-900 dark:text-white">
            ${cls.assessment ? "Feedback" : "Give Feedback"}
          </p>
          <form id="feedbackForm" data-class-id="${cls.id}" class="space-y-3">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Rating (0–10)</label>
              <input type="number" name="rating" min="0" max="10" step="0.1" required
                value="${a.rating ?? ""}"
                class="block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Grammar Corrections</label>
              <textarea name="grammar_corrections" rows="3"
                class="tinymce-field block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">${escapeHtml(a.grammar_corrections || "")}</textarea>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Recommendation</label>
              <textarea name="recommendation" rows="3"
                class="tinymce-field block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">${escapeHtml(a.recommendation || "")}</textarea>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Homework Description</label>
              <textarea name="homework" rows="3"
                class="tinymce-field block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">${escapeHtml(a.homework || "")}</textarea>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Remarks</label>
              <textarea name="remarks" rows="3"
                class="tinymce-field block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">${escapeHtml(a.remarks || "")}</textarea>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">Homework Title (optional attachment)</label>
              <input type="text" name="homework_title"
                value="${escapeHtml(a.homework_title || "")}"
                placeholder="e.g. Unit 3 Worksheet"
                class="block w-full rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-gray-900 focus:ring-2 focus:ring-gray-900/10 dark:border-gray-700 dark:bg-gray-800 dark:text-white">
            </div>
            <div class="flex justify-end">
              <button type="submit" id="feedbackSubmitBtn"
                class="rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-blue-700 disabled:opacity-60">
                ${cls.assessment ? "Update Feedback" : "Save Feedback"}
              </button>
            </div>
          </form>
        </div>`;
    }

    detailBody.innerHTML = infoHtml + attendanceHtml + feedbackHtml;
    initFeedbackEditors();
  }

  function openDetailModal(classId) {
    const cls = state.classes.find((c) => c.id === classId);
    if (!cls) return;
    detailClassId = classId;
    renderDetailBody(cls);
    detailModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeDetailModal() {
    destroyFeedbackEditors();
    detailModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  function showRefundPrompt(classId) {
    const container = detailBody.querySelector("[data-attendance='absent']")
      ?.parentElement?.parentElement;
    if (!container) return;
    container.innerHTML = `
      <p class="mb-3 text-sm font-medium text-gray-900 dark:text-white">Refund one class credit for this absence?</p>
      <div class="flex gap-2">
        <button type="button" data-attendance="absent" data-class-id="${classId}" data-refund="1"
          class="flex-1 rounded-xl bg-blue-600 px-3 py-2.5 text-sm font-medium text-white transition hover:bg-blue-700">
          Refund
        </button>
        <button type="button" data-attendance="absent" data-class-id="${classId}" data-refund="0"
          class="flex-1 rounded-xl border border-gray-200 px-3 py-2.5 text-sm font-medium text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          No Refund
        </button>
        <button type="button" data-cancel-absent
          class="rounded-xl border border-gray-200 px-3 py-2.5 text-sm text-gray-500 transition hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800">
          Cancel
        </button>
      </div>`;
  }

  async function applyAttendance(classId, status, refund) {
    try {
      const body = new URLSearchParams({
        action: "set_attendance",
        class_id: String(classId),
        status,
        refund: refund ? "1" : "0",
      });
      const res = await fetch(scheduleUrl, { method: "POST", body });
      const data = await res.json();
      if (data.status !== "ok")
        throw new Error(data.message || "Could not tag attendance.");

      await loadSchedule();
      openDetailModal(classId);
    } catch (err) {
      alert(err.message || "Could not tag attendance.");
    }
  }

  detailBody.addEventListener("click", (e) => {
    const att = e.target.closest("[data-attendance]");
    if (att) {
      const classId = Number(att.dataset.classId || detailClassId);
      const status = att.dataset.attendance;
      if (status === "absent" && att.dataset.refund === undefined) {
        showRefundPrompt(classId);
        return;
      }
      applyAttendance(classId, status, att.dataset.refund === "1");
      return;
    }
    if (e.target.closest("[data-cancel-absent]")) {
      openDetailModal(detailClassId);
    }
  });

  detailBody.addEventListener("submit", async (e) => {
    if (e.target.id !== "feedbackForm") return;
    e.preventDefault();

    const btn = e.target.querySelector("#feedbackSubmitBtn");
    if (btn) btn.disabled = true;

    try {
      // TinyMCE only writes back to its textarea's .value on demand, not on every
      // keystroke — without this, FormData below would read stale (initial) content.
      if (typeof tinymce !== "undefined") tinymce.triggerSave();

      const fd = new FormData(e.target);
      fd.set("action", "save_feedback");
      fd.set("class_id", String(detailClassId));

      const res = await fetch(scheduleUrl, { method: "POST", body: fd });
      const data = await res.json();
      if (data.status !== "ok")
        throw new Error(data.message || "Could not save feedback.");

      await loadSchedule();
      openDetailModal(detailClassId);
    } catch (err) {
      alert(err.message || "Could not save feedback.");
      if (btn) btn.disabled = false;
    }
  });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    submitBtn.disabled = true;
    try {
      const response = await fetch(scheduleUrl, {
        method: "POST",
        body: new FormData(form),
      });
      const data = await response.json();
      if (data.status === "ok") {
        closeModal();
        await loadSchedule();
      } else {
        alert("Error: " + (data.message || "Could not schedule the class."));
      }
    } catch (error) {
      console.error("Error scheduling class:", error);
      alert("Could not schedule the class. Please try again.");
    } finally {
      submitBtn.disabled = enrollmentSelect.options.length <= 1;
    }
  });

  enrollmentSelect.addEventListener("change", updateClassPreview);
  classStartInput.addEventListener("input", updateClassPreview);

  cellsEl.addEventListener("click", (e) => {
    const block = e.target.closest(".calendar-event");
    if (block) {
      e.stopPropagation();
      openDetailModal(Number(block.dataset.classId));
      return;
    }
    const cell = e.target.closest("[data-hour]");
    if (cell) openAddModal(Number(cell.dataset.hour));
  });

  cellsEl.addEventListener("keydown", (e) => {
    if (e.key !== "Enter" && e.key !== " ") return;
    if (e.target.closest(".calendar-event")) return;
    const cell = e.target.closest("[data-hour]");
    if (!cell) return;
    e.preventDefault();
    openAddModal(Number(cell.dataset.hour));
  });

  monthGrid.addEventListener("click", (e) => {
    const btn = e.target.closest("[data-date]");
    if (!btn) return;
    state.view = "day";
    applyViewButtonStyles();
    goToDate(parseLocalDate(btn.dataset.date));
  });

  modal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-modal]")) closeModal();
  });
  detailModal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-detail]")) closeDetailModal();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key !== "Escape") return;
    if (!modal.classList.contains("hidden")) closeModal();
    if (!detailModal.classList.contains("hidden")) closeDetailModal();
  });

  viewDayBtn.addEventListener("click", () => setView("day"));
  viewMonthBtn.addEventListener("click", () => setView("month"));

  document
    .getElementById("calendarPrev")
    .addEventListener("click", () => shiftDate(-1));
  document
    .getElementById("calendarNext")
    .addEventListener("click", () => shiftDate(1));
  document
    .getElementById("calendarToday")
    .addEventListener("click", () => goToDate(new Date()));

  dateInput.addEventListener("change", () => {
    if (!dateInput.value) return;
    goToDate(parseLocalDate(dateInput.value));
  });

  buildGridSkeleton();
  applyViewButtonStyles();
  refreshDateControls();
  loadSchedule();
})();
