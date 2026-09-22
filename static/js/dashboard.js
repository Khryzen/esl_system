// dashboard.js: the day-view class calendar on the dashboard.
//
// GET  <page url>?start=YYYY-MM-DD&days=1   -> { status, classes: [...], enrollments: [...] }
// POST <page url>  enrollment_id, date, start_time -> { status, class_id, classes_remaining }
// See DashboardHandler (dashboard_handler.go) for the exact shape of both.

(() => {
  "use strict";

  // The grid covers 7:00 AM through 9:00 PM. Change both to widen the day, and keep
  // ROW_HEIGHT in sync with the inline `height` set on each row in buildGridSkeleton.
  const START_HOUR = 7;
  const END_HOUR = 21;
  const ROW_HEIGHT = 64; // px, one hour tall

  const scheduleUrl = window.location.pathname; // GET and POST both hit this page's own route

  const loadingEl = document.getElementById("calendarLoading");
  const scrollEl = document.getElementById("calendarScroll");
  const labelsEl = document.getElementById("calendarLabels");
  const cellsEl = document.getElementById("calendarCells");
  const dateLabelEl = document.getElementById("calendarDateLabel");
  const dateInput = document.getElementById("calendarDateInput");

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

  if (!scrollEl || !cellsEl) return; // calendar isn't on this page

  const state = {
    date: startOfDay(new Date()),
    classes: [],
    enrollments: [],
  };

  /* ---------- Small date/time helpers ---------- */

  function startOfDay(d) {
    return new Date(d.getFullYear(), d.getMonth(), d.getDate());
  }

  function pad2(n) {
    return String(n).padStart(2, "0");
  }

  // Local YYYY-MM-DD, matching what the Go handler expects and what <input type="date"> uses.
  function toDateStr(d) {
    return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
  }

  // The server sends "YYYY-MM-DDTHH:MM:SS" with no timezone suffix, so the browser
  // parses it as local time, matching the local time.Local the server used to format it.
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
    const d = new Date(2000, 0, 1, h, m + minutes);
    return d;
  }

  /* ---------- Grid skeleton (built once) ---------- */

  function buildGridSkeleton() {
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

  /* ---------- Rendering classes already on the calendar ---------- */

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
      if (end <= gridStart || start >= gridEnd) continue; // outside the visible hours

      const top = Math.max(0, (start - gridStart) / 60000) * pxPerMinute;
      const bottom = Math.min(
        gridHeight,
        ((Math.max(end, start) - gridStart) / 60000) * pxPerMinute,
      );

      const block = document.createElement("div");
      block.className =
        "calendar-event pointer-events-none absolute inset-x-2 overflow-hidden rounded-lg border-l-4 border-blue-500 bg-blue-50 px-2 py-1 text-xs leading-tight text-blue-900 shadow-sm dark:border-blue-400 dark:bg-blue-950/50 dark:text-blue-100";
      block.style.top = `${top}px`;
      block.style.height = `${Math.max(bottom - top, 18)}px`;
      block.title = `${cls.student} — ${cls.course}\n${formatClock(start)}–${formatClock(end)}`;
      block.innerHTML = `
        <p class="truncate font-medium">${escapeHtml(cls.student)}</p>
        <p class="truncate text-blue-700/80 dark:text-blue-200/80">${escapeHtml(cls.course)}</p>
      `;
      cellsEl.appendChild(block);
    }
  }

  function escapeHtml(s) {
    const div = document.createElement("div");
    div.textContent = s ?? "";
    return div.innerHTML;
  }

  /* ---------- Fetching the day's schedule ---------- */

  async function loadSchedule() {
    loadingEl.textContent = "Loading schedule…";
    loadingEl.classList.remove("hidden");
    scrollEl.classList.add("hidden");

    try {
      const url = `${scheduleUrl}?start=${toDateStr(state.date)}&days=1`;
      // POST, not GET: a GET to this page's own route re-renders the full page (like
      // every other page here); POST is treated as an action and returns JSON instead,
      // the same way the create-class request below already does.
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

      state.classes = data.classes || [];
      state.enrollments = data.enrollments || [];
      renderEvents();

      loadingEl.classList.add("hidden");
      scrollEl.classList.remove("hidden");
    } catch (error) {
      console.error("Error loading schedule:", error);
      loadingEl.textContent = "Could not load the schedule. Try again.";
    }
  }

  function refreshDateControls() {
    dateInput.value = toDateStr(state.date);
    dateLabelEl.textContent = state.date.toLocaleDateString(undefined, {
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

  // Duration and end time are read-only: they always follow the selected package.
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
    const cell = e.target.closest("[data-hour]");
    if (cell) openAddModal(Number(cell.dataset.hour));
  });

  cellsEl.addEventListener("keydown", (e) => {
    if (e.key !== "Enter" && e.key !== " ") return;
    const cell = e.target.closest("[data-hour]");
    if (!cell) return;
    e.preventDefault();
    openAddModal(Number(cell.dataset.hour));
  });

  modal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-modal]")) closeModal();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !modal.classList.contains("hidden")) closeModal();
  });

  document.getElementById("calendarPrevDay").addEventListener("click", () => {
    const d = new Date(state.date);
    d.setDate(d.getDate() - 1);
    goToDate(d);
  });

  document.getElementById("calendarNextDay").addEventListener("click", () => {
    const d = new Date(state.date);
    d.setDate(d.getDate() + 1);
    goToDate(d);
  });

  document
    .getElementById("calendarToday")
    .addEventListener("click", () => goToDate(new Date()));

  dateInput.addEventListener("change", () => {
    if (!dateInput.value) return;
    // Parsed as local midnight, matching startOfDay elsewhere.
    const [y, m, d] = dateInput.value.split("-").map(Number);
    goToDate(new Date(y, m - 1, d));
  });

  /* ---------- Init ---------- */

  buildGridSkeleton();
  refreshDateControls();
  loadSchedule();
})();
