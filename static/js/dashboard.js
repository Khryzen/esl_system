// dashboard.js: the day/month class calendar on the dashboard.
//
// POST <page url>?start=YYYY-MM-DD&days=N             -> { status, view:"day", classes, enrollments }
// POST <page url>?start=YYYY-MM-DD&days=N&view=month  -> { status, view:"month", counts }
// POST <page url>  enrollment_id, date, start_time    -> { status, class_id, classes_remaining }

(() => {
  "use strict";

  // Grid covers the full day. 24 rows of 64px = 1536px tall, scrollable.
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

  if (!scrollEl || !cellsEl) return; // calendar isn't on this page

  const state = {
    view: "day", // "day" | "month"
    date: startOfDay(new Date()),
    classes: [],
    enrollments: [],
    counts: {},
  };

  /* ---------- Small date/time helpers ---------- */

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
    const period = hour < 12 ? "AM" : "PM";
    const h12 = hour % 12 === 0 ? 12 : hour % 12;
    return `${h12}:00 ${period}`;
  }
  function formatClock(d) {
    const period = d.getHours() < 12 ? "AM" : "PM";
    const h12 = d.getHours() % 12 === 0 ? 12 : d.getHours() % 12;
    return `${h12}:${pad2(d.getMinutes())} ${period}`;
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

  /* ---------- Grid skeleton (built once) ---------- */

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

  /* ---------- Day view rendering ---------- */

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
      block.className =
        "calendar-event cursor-pointer absolute inset-x-2 overflow-hidden rounded-lg border-l-4 border-blue-500 bg-blue-50 px-2 py-1 text-xs leading-tight text-blue-900 shadow-sm transition hover:bg-blue-100 dark:border-blue-400 dark:bg-blue-950/50 dark:text-blue-100 dark:hover:bg-blue-950/80";
      block.style.top = `${top}px`;
      block.style.height = `${Math.max(bottom - top, 18)}px`;
      block.dataset.classId = String(cls.id);
      block.title = `${cls.student} — ${cls.course}\n${formatClock(start)}–${formatClock(end)}`;
      block.innerHTML = `
        <p class="truncate font-medium">${escapeHtml(cls.student)}</p>
        <p class="truncate text-blue-700/80 dark:text-blue-200/80">${escapeHtml(cls.course)}</p>
      `;
      cellsEl.appendChild(block);
    }
  }

  /* ---------- Month view rendering ---------- */

  function monthGridStart(d) {
    const first = new Date(d.getFullYear(), d.getMonth(), 1);
    const start = new Date(first);
    start.setDate(first.getDate() - first.getDay()); // back to Sunday
    return startOfDay(start);
  }

  function renderMonth() {
    monthGrid.innerHTML = "";

    const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    for (const wd of weekdays) {
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
      if (isToday) {
        num.className =
          "flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white";
      } else {
        num.className =
          "text-sm " +
          (inMonth
            ? "text-gray-900 dark:text-white"
            : "text-gray-400 dark:text-gray-600");
      }
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

  /* ---------- Fetching ---------- */

  async function loadSchedule() {
    loadingEl.textContent = "Loading schedule…";
    loadingEl.classList.remove("hidden");
    scrollEl.classList.add("hidden");
    monthEl.classList.add("hidden");

    try {
      let url;
      if (state.view === "month") {
        const start = monthGridStart(state.date);
        url = `${scheduleUrl}?start=${toDateStr(start)}&days=42&view=month`;
      } else {
        url = `${scheduleUrl}?start=${toDateStr(state.date)}&days=1`;
      }

      const response = await fetch(url, { method: "POST" });

      const contentType = response.headers.get("content-type") || "";
      if (!contentType.includes("application/json")) {
        throw new Error(
          `Server returned ${response.status} instead of the schedule.`,
        );
      }

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
        scrollEl.scrollTop = 7 * ROW_HEIGHT; // land on working hours, not midnight
      }

      loadingEl.classList.add("hidden");
    } catch (error) {
      console.error("Error loading schedule:", error);
      loadingEl.textContent = "Could not load the schedule. Try again.";
    }
  }

  /* ---------- View toggle ---------- */

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

  /* ---------- Date controls ---------- */

  function refreshDateControls() {
    dateInput.value = toDateStr(state.date);
    if (state.view === "month") {
      dateLabelEl.textContent = state.date.toLocaleDateString(undefined, {
        month: "long",
        year: "numeric",
      });
    } else {
      dateLabelEl.textContent = state.date.toLocaleDateString(undefined, {
        weekday: "long",
        year: "numeric",
        month: "long",
        day: "numeric",
      });
    }
  }

  function goToDate(date) {
    state.date = startOfDay(date);
    refreshDateControls();
    loadSchedule();
  }

  function shiftDate(delta) {
    const d = new Date(state.date);
    if (state.view === "month") {
      d.setMonth(d.getMonth() + delta);
    } else {
      d.setDate(d.getDate() + delta);
    }
    goToDate(d);
  }

  /* ---------- Add-class modal ---------- */

  function populateEnrollmentSelect() {
    enrollmentSelect.innerHTML =
      '<option value="" disabled selected>-- Select an enrollment --</option>';

    const hasEnrollments = state.enrollments.length > 0;
    enrollmentEmpty.classList.toggle("hidden", hasEnrollments);
    enrollmentSelect.disabled = !hasEnrollments;
    submitBtn.disabled = !hasEnrollments;

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

  /* ---------- Class details modal ---------- */

  function openDetailModal(classId) {
    const cls = state.classes.find((c) => c.id === classId);
    if (!cls) return;

    const start = parseServerTime(cls.start);
    const end = parseServerTime(cls.end);

    const rows = [
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
      ["Status", cls.present ? "Present" : "Scheduled"],
    ];

    detailBody.innerHTML = rows
      .map(
        ([label, value]) => `
        <div class="flex justify-between gap-4 border-b border-gray-100 pb-2 last:border-0 dark:border-gray-800">
          <span class="text-gray-500 dark:text-gray-400">${escapeHtml(label)}</span>
          <span class="text-right font-medium text-gray-900 dark:text-white">${escapeHtml(String(value))}</span>
        </div>`,
      )
      .join("");

    detailModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeDetailModal() {
    detailModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  /* ---------- Form submit ---------- */

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

  /* ---------- Event wiring ---------- */

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

  /* ---------- Init ---------- */

  buildGridSkeleton();
  applyViewButtonStyles();
  refreshDateControls();
  loadSchedule();
})();
