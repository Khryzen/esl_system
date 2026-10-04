(() => {
  "use strict";

  const START_HOUR = 0;
  const END_HOUR = 24;
  const ROW_HEIGHT = 72;
  const pageUrl = window.location.pathname;
  const loadingEl = document.getElementById("calendarLoading");
  const scrollEl = document.getElementById("calendarScroll");
  const monthEl = document.getElementById("calendarMonth");
  const monthGrid = document.getElementById("monthGrid");
  const labelsEl = document.getElementById("calendarLabels");
  const cellsEl = document.getElementById("calendarCells");
  const dateLabelEl = document.getElementById("calendarDateLabel");
  const dateInput = document.getElementById("calendarDateInput");
  const viewDayBtn = document.getElementById("viewDayBtn");
  const viewMonthBtn = document.getElementById("viewMonthBtn");
  const prevBtn = document.getElementById("calendarPrev");
  const todayBtn = document.getElementById("calendarToday");
  const nextBtn = document.getElementById("calendarNext");
  const newClassButton = document.getElementById("newClassButton");
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
  if (
    !loadingEl ||
    !scrollEl ||
    !cellsEl ||
    !monthEl ||
    !monthGrid ||
    !modal ||
    !detailModal
  ) {
    return;
  }

  const state = {
    view: "day",
    date: startOfDay(new Date()),
    classes: [],
    enrollments: [],
    counts: {},
  };
  let detailClassId = 0;
  /* =====================================================
    DATE HELPERS
  ====================================================== */
  function startOfDay(date) {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate());
  }
  function pad2(value) {
    return String(value).padStart(2, "0");
  }
  function toDateString(date) {
    return [
      date.getFullYear(),
      pad2(date.getMonth() + 1),
      pad2(date.getDate()),
    ].join("-");
  }
  function parseLocalDate(value) {
    const [year, month, day] = value.split("-").map(Number);
    return new Date(year, month - 1, day);
  }
  function parseServerTime(value) {
    return new Date(value);
  }
  function formatHour(hour) {
    const suffix = hour < 12 ? "AM" : "PM";
    const displayHour = hour % 12 === 0 ? 12 : hour % 12;
    return `${displayHour}:00 ${suffix}`;
  }
  function formatClock(date) {
    const suffix = date.getHours() < 12 ? "AM" : "PM";
    const hour = date.getHours() % 12 || 12;
    return `${hour}:${pad2(date.getMinutes())} ${suffix}`;
  }

  function addMinutes(time, minutes) {
    const [hour, minute] = time.split(":").map(Number);
    return new Date(2000, 0, 1, hour, minute + minutes);
  }
  function escapeHtml(value) {
    const div = document.createElement("div");
    div.textContent = value ?? "";
    return div.innerHTML;
  }

  /* =====================================================
  CALENDAR
  ====================================================== */

  function classColors(status) {
    if (status === "present") {
      return [
        "border-green-500",
        "bg-green-50",
        "text-green-900",
        "dark:border-green-400",
        "dark:bg-green-950/50",
        "dark:text-green-100",
      ].join(" ");
    }
    if (status === "absent") {
      return [
        "border-red-500",
        "bg-red-50",
        "text-red-900",
        "dark:border-red-400",
        "dark:bg-red-950/50",
        "dark:text-red-100",
      ].join(" ");
    }
    return [
      "border-blue-500",
      "bg-blue-50",
      "text-blue-900",
      "dark:border-blue-400",
      "dark:bg-blue-950/50",
      "dark:text-blue-100",
    ].join(" ");
  }
  function buildGridSkeleton() {
    const height = (END_HOUR - START_HOUR) * ROW_HEIGHT;
    cellsEl.style.height = `${height}px`;
    labelsEl.innerHTML = "";
    cellsEl.innerHTML = "";
    for (let hour = START_HOUR; hour < END_HOUR; hour++) {
      const label = document.createElement("div");
      label.className =
        "flex items-start justify-end pr-3 pt-0 -translate-y-2.5 text-xs text-slate-400 dark:text-slate-500";
      label.style.height = `${ROW_HEIGHT}px`;
      label.textContent = formatHour(hour);
      labelsEl.appendChild(label);
      const cell = document.createElement("div");
      cell.className =
        "calendar-hour cursor-pointer border-b border-slate-100 transition hover:bg-blue-50/60 dark:border-slate-800 dark:hover:bg-blue-950/20";
      cell.dataset.hour = String(hour);
      cell.setAttribute("role", "button");
      cell.setAttribute("tabindex", "0");
      cell.setAttribute(
        "aria-label",
        `Schedule a class at ${formatHour(hour)}`,
      );
      cellsEl.appendChild(cell);
    }
  }

  function clearEvents() {
    cellsEl
      .querySelectorAll(".calendar-event")
      .forEach((element) => element.remove());
  }

  function renderEvents() {
    clearEvents();
    const gridStart = new Date(state.date);
    gridStart.setHours(START_HOUR, 0, 0, 0);
    const gridEnd = new Date(state.date);
    gridEnd.setHours(END_HOUR, 0, 0, 0);
    const gridHeight = (END_HOUR - START_HOUR) * ROW_HEIGHT;
    const pixelsPerMinute = ROW_HEIGHT / 60;
    for (const cls of state.classes) {
      const start = parseServerTime(cls.start);
      const end = parseServerTime(cls.end);
      if (end <= gridStart || start >= gridEnd) {
        continue;
      }
      const top = Math.max(0, (start - gridStart) / 60000) * pixelsPerMinute;
      const bottom = Math.min(
        gridHeight,
        ((Math.max(end, start) - gridStart) / 60000) * pixelsPerMinute,
      );
      const block = document.createElement("button");
      block.type = "button";
      block.className = `calendar-event ${classColors(cls.status)}`;
      block.style.top = `${top}px`;
      block.style.height = `${Math.max(bottom - top, 22)}px`;
      block.dataset.classId = String(cls.id);
      block.title = `${cls.student} — ${cls.course}`;
      block.innerHTML = `
        <p class="truncate text-left font-semibold">
          ${escapeHtml(cls.student)}
        </p>
        <p class="truncate text-left opacity-75">
          ${escapeHtml(cls.course)}
        </p>
        <p class="truncate text-left opacity-60">
          ${formatClock(start)}–${formatClock(end)}
        </p>
      `;
      cellsEl.appendChild(block);
    }
  }

  function monthGridStart(date) {
    const first = new Date(date.getFullYear(), date.getMonth(), 1);
    const start = new Date(first);
    start.setDate(first.getDate() - first.getDay());
    return startOfDay(start);
  }

  function renderMonth() {
    monthGrid.innerHTML = "";
    const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    for (const weekday of weekdays) {
      const header = document.createElement("div");
      header.className =
        "bg-slate-50 py-2.5 text-center text-xs font-semibold text-slate-500 dark:bg-slate-800/70 dark:text-slate-400";
      header.textContent = weekday;
      monthGrid.appendChild(header);
    }
    const start = monthGridStart(state.date);
    const today = startOfDay(new Date());
    for (let i = 0; i < 42; i++) {
      const date = new Date(start);
      date.setDate(start.getDate() + i);
      const key = toDateString(date);
      const count = state.counts[key] || 0;
      const button = document.createElement("button");
      button.type = "button";
      button.dataset.date = key;
      button.className =
        "flex min-h-[100px] flex-col gap-2 bg-white p-2.5 text-left transition hover:bg-blue-50 dark:bg-slate-900 dark:hover:bg-blue-950/20";
      const inMonth = date.getMonth() === state.date.getMonth();
      const isToday = date.getTime() === today.getTime();
      const number = document.createElement("span");
      number.className = isToday
        ? "flex h-7 w-7 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white"
        : "text-sm font-medium " +
          (inMonth
            ? "text-slate-900 dark:text-white"
            : "text-slate-400 dark:text-slate-600");
      number.textContent = String(date.getDate());
      button.appendChild(number);
      if (count > 0) {
        const badge = document.createElement("span");
        badge.className =
          "self-start rounded-full bg-blue-100 px-2 py-1 text-[10px] font-semibold text-blue-700 dark:bg-blue-950/60 dark:text-blue-200";
        badge.textContent = `${count} class${count === 1 ? "" : "es"}`;
        button.appendChild(badge);
      }
      monthGrid.appendChild(button);
    }
  }

  /* =====================================================
    SERVER
  ====================================================== */
  async function loadSchedule() {
    loadingEl.classList.remove("hidden");
    scrollEl.classList.add("hidden");
    monthEl.classList.add("hidden");
    try {
      const params = new URLSearchParams();
      if (state.view === "month") {
        params.set("start", toDateString(monthGridStart(state.date)));
        params.set("days", "42");
        params.set("view", "month");
      } else {
        params.set("start", toDateString(state.date));
        params.set("days", "1");
      }
      const response = await fetch(`${pageUrl}?${params.toString()}`, {
        method: "POST",
      });
      const contentType = response.headers.get("content-type") || "";
      if (!contentType.includes("application/json")) {
        throw new Error(`Server returned ${response.status}.`);
      }
      const data = await response.json();
      if (data.status !== "ok") {
        throw new Error(data.message || "Could not load the schedule.");
      }
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
      if (state.view === "day") {
        populateEnrollmentSelect();
      }
    } catch (error) {
      console.error("Error loading schedule:", error);
      loadingEl.innerHTML = `
        <div class="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full bg-red-50 text-red-500 dark:bg-red-950/40 dark:text-red-400">
          <i data-lucide="triangle-alert" class="h-5 w-5"></i>
        </div>
        <p class="text-sm font-medium text-slate-700 dark:text-slate-300">
          Could not load the schedule.
        </p>
        <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
          ${escapeHtml(error.message)}
        </p>
      `;
      if (window.lucide) {
        window.lucide.createIcons();
      }
    }
  }

  /* =====================================================
    VIEW CONTROLS
  ====================================================== */

  function applyViewButtonStyles() {
    const active = "bg-blue-600 text-white shadow-sm";
    const inactive =
      "text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700";
    viewDayBtn.className = `rounded-lg px-3 py-1.5 text-sm font-medium transition ${
      state.view === "day" ? active : inactive
    }`;
    viewMonthBtn.className = `rounded-lg px-3 py-1.5 text-sm font-medium transition ${
      state.view === "month" ? active : inactive
    }`;
  }

  function refreshDateControls() {
    dateInput.value = toDateString(state.date);
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

  function setView(view) {
    if (state.view === view) {
      return;
    }
    state.view = view;
    applyViewButtonStyles();
    refreshDateControls();
    loadSchedule();
  }
  function goToDate(date) {
    state.date = startOfDay(date);
    refreshDateControls();
    loadSchedule();
  }
  function shiftDate(delta) {
    const date = new Date(state.date);
    if (state.view === "month") {
      date.setMonth(date.getMonth() + delta);
    } else {
      date.setDate(date.getDate() + delta);
    }
    goToDate(date);
  }

  /* =====================================================
    SCHEDULE MODAL
  ====================================================== */

  function populateEnrollmentSelect() {
    enrollmentSelect.innerHTML = `
        <option value="" disabled selected>
          -- Select an enrollment --
        </option>
      `;
    const hasEnrollments = state.enrollments.length > 0;
    enrollmentEmpty.classList.toggle("hidden", hasEnrollments);
    enrollmentSelect.disabled = !hasEnrollments;
    submitBtn.disabled = !hasEnrollments;
    for (const enrollment of state.enrollments) {
      const option = document.createElement("option");
      option.value = enrollment.id;
      option.textContent = `${enrollment.student} — ${enrollment.course} (${enrollment.package}), ${enrollment.classes_remaining} left`;
      option.dataset.durationMinutes = enrollment.duration_minutes;
      option.dataset.classesRemaining = enrollment.classes_remaining;
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
    remainingEl.textContent = `${remaining} class${
      remaining === 1 ? "" : "es"
    } remaining on this enrollment`;
  }

  function openAddModal(hour = null) {
    form.reset();
    populateEnrollmentSelect();
    classDateInput.value = toDateString(state.date);
    if (hour !== null) {
      classStartInput.value = `${pad2(hour)}:00`;
    }
    updateScheduleDateTimeLimits();
    updateClassPreview();
    modal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
    enrollmentSelect.focus();
  }
  function closeModal() {
    modal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  /* =====================================================
    FEEDBACK EDITOR
  ====================================================== */
  const TINYMCE_SELECTOR = "#classDetailBody textarea.tinymce-field";
  function initFeedbackEditors() {
    if (typeof tinymce === "undefined") {
      return;
    }
    tinymce.init({
      selector: TINYMCE_SELECTOR,
      height: 150,
      menubar: false,
      statusbar: false,
      branding: false,
      plugins: "lists",
      toolbar: "bold italic underline | bullist numlist | removeformat",
    });
  }

  function destroyFeedbackEditors() {
    if (typeof tinymce === "undefined") {
      return;
    }
    tinymce.remove(TINYMCE_SELECTOR);
  }

  /* =====================================================
    CLASS DETAILS
  ====================================================== */
  function renderDetailBody(cls) {
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
      <div class="rounded-2xl border border-slate-200 dark:border-slate-800">
        <div class="border-b border-slate-100 px-4 py-3 dark:border-slate-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-white">
            Class Information
          </h3>
        </div>
        <div class="divide-y divide-slate-100 dark:divide-slate-800">
          ${infoRows
            .map(
              ([label, value]) => `
                <div class="flex justify-between gap-4 px-4 py-3">
                  <span class="text-sm text-slate-500 dark:text-slate-400">
                    ${escapeHtml(label)}
                  </span>

                  <span class="text-right text-sm font-medium text-slate-900 dark:text-white">
                    ${escapeHtml(String(value))}
                  </span>
                </div>
              `,
            )
            .join("")}
        </div>
      </div>
    `;
    let attendanceHtml = "";
    if (!cls.status) {
      attendanceHtml = `
        <div class="rounded-2xl border border-slate-200 p-4 dark:border-slate-800">
          <div class="mb-3">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-white">
              Attendance
            </h3>
            <p class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
              Record whether the student attended this class.
            </p>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <button
              type="button"
              data-attendance="present"
              data-class-id="${cls.id}"
              class="class-button inline-flex w-full items-center justify-center gap-2 bg-green-600 text-white hover:bg-green-700"
            >
              <i data-lucide="check" class="h-4 w-4 shrink-0"></i>
              <span>Present</span>
            </button>
            <button
              type="button"
              data-attendance="absent"
              data-class-id="${cls.id}"
              class="class-button inline-flex w-full items-center justify-center gap-2 bg-red-600 text-white hover:bg-red-700"
            >
              <i data-lucide="x" class="h-4 w-4 shrink-0"></i>
              <span>Absent</span>
            </button>
          </div>
        </div>
      `;
    } else if (cls.status === "present") {
      attendanceHtml = `
        <div class="flex items-center justify-between rounded-2xl border border-green-200 bg-green-50 px-4 py-4 dark:border-green-900/60 dark:bg-green-950/40">
          <div>
            <p class="text-sm font-semibold text-green-900 dark:text-green-200">
              Attendance recorded
            </p>
            <p class="mt-0.5 text-xs text-green-700 dark:text-green-300">
              The student was marked present.
            </p>
          </div>
          <span class="rounded-full bg-green-600 px-3 py-1 text-xs font-semibold text-white">
            Present
          </span>
        </div>
      `;
    } else {
      attendanceHtml = `
        <div class="rounded-2xl border border-red-200 bg-red-50 px-4 py-4 dark:border-red-900/60 dark:bg-red-950/40">
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-sm font-semibold text-red-900 dark:text-red-200">
                Attendance recorded
              </p>
              <p class="mt-0.5 text-xs text-red-700 dark:text-red-300">
                ${
                  cls.credit_refunded
                    ? "The class credit was refunded."
                    : "No class credit was refunded."
                }
              </p>
            </div>
            <span class="rounded-full bg-red-600 px-3 py-1 text-xs font-semibold text-white">
              Absent
            </span>
          </div>
        </div>
      `;
    }

    let feedbackHtml = "";
    if (cls.status === "present") {
      const assessment = cls.assessment || {};
      feedbackHtml = `
        <div class="rounded-2xl border border-slate-200 p-4 dark:border-slate-800">
          <div class="mb-4">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-white">
              ${cls.assessment ? "Student Feedback" : "Record Feedback"}
            </h3>
            <p class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
              Record the student's performance and follow-up notes.
            </p>
          </div>
          <form
            id="feedbackForm"
            data-class-id="${cls.id}"
            class="space-y-4"
          >
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Rating (0–10)
              </label>
              <input
                type="number"
                name="rating"
                min="0"
                max="10"
                step="0.1"
                required
                value="${assessment.rating ?? ""}"
                  class="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"

              />
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Grammar Corrections
              </label>
              <textarea
                name="grammar_corrections"
                rows="3"
                class="tinymce-field class-input"
              >${escapeHtml(assessment.grammar_corrections || "")}</textarea>
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Recommendation
              </label>
              <textarea
                name="recommendation"
                rows="3"
                class="tinymce-field class-input"
              >${escapeHtml(assessment.recommendation || "")}</textarea>
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Homework Description
              </label>
              <textarea
                name="homework"
                rows="3"
                class="tinymce-field class-input"
              >${escapeHtml(assessment.homework || "")}</textarea>
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Remarks
              </label>
              <textarea
                name="remarks"
                rows="3"
                class="tinymce-field class-input"
              >${escapeHtml(assessment.remarks || "")}</textarea>
            </div>
            <div>
              <label class="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-400">
                Homework Title
              </label>
              <input
                type="text"
                name="homework_title"
                value="${escapeHtml(assessment.homework_title || "")}"
                placeholder="e.g. Unit 3 Worksheet"
                  class="w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-700 dark:bg-slate-900 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"

              />
            </div>
            <div class="flex justify-end pt-2">
              <button
                type="submit"
                id="feedbackSubmitBtn"
                class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-blue-700 disabled:opacity-60"
              >
                <i data-lucide="save" class="h-4 w-4"></i>
                ${cls.assessment ? "Update Feedback" : "Save Feedback"}
              </button>
            </div>
          </form>
        </div>
      `;
    }
    detailBody.innerHTML = infoHtml + attendanceHtml + feedbackHtml;
    if (window.lucide) {
      window.lucide.createIcons();
    }
    initFeedbackEditors();
  }

  function openDetailModal(classId) {
    const cls = state.classes.find((item) => item.id === classId);
    if (!cls) {
      return;
    }
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

  /* =====================================================
    ATTENDANCE
  ====================================================== */

  function showRefundPrompt(classId) {
    const attendanceButton = detailBody.querySelector(
      "[data-attendance='absent']",
    );

    const container = attendanceButton?.parentElement?.parentElement;

    if (!container) {
      return;
    }

    container.innerHTML = `
      <p class="mb-3 text-sm font-semibold text-slate-900 dark:text-white">
        Refund one class credit for this absence?
      </p>

      <div class="grid grid-cols-3 gap-2">

        <button
          type="button"
          data-attendance="absent"
          data-class-id="${classId}"
          data-refund="1"
          class="class-button bg-blue-600 text-white hover:bg-blue-700"
        >
          Refund
        </button>

        <button
          type="button"
          data-attendance="absent"
          data-class-id="${classId}"
          data-refund="0"
          class="class-button-secondary"
        >
          No Refund
        </button>

        <button
          type="button"
          data-cancel-absent
          class="class-button-secondary"
        >
          Cancel
        </button>

      </div>
    `;
  }

  async function applyAttendance(classId, status, refund) {
    try {
      const body = new URLSearchParams({
        action: "set_attendance",
        class_id: String(classId),
        status,
        refund: refund ? "1" : "0",
      });
      const response = await fetch(pageUrl, {
        method: "POST",
        body,
      });
      const data = await response.json();
      if (data.status !== "ok") {
        throw new Error(data.message || "Could not tag attendance.");
      }
      await loadSchedule();
      openDetailModal(classId);
    } catch (error) {
      Swal.fire({
        icon: "error",
        title: "Attendance Error",
        text: error.message || "Could not tag attendance.",
      });
    }
  }

  /* =====================================================
    EVENT HANDLERS
  ====================================================== */

  detailBody.addEventListener("click", (event) => {
    const attendance = event.target.closest("[data-attendance]");
    if (attendance) {
      const classId = Number(attendance.dataset.classId || detailClassId);
      const status = attendance.dataset.attendance;
      if (status === "absent" && attendance.dataset.refund === undefined) {
        showRefundPrompt(classId);
        return;
      }
      applyAttendance(classId, status, attendance.dataset.refund === "1");
      return;
    }
    if (event.target.closest("[data-cancel-absent]")) {
      openDetailModal(detailClassId);
    }
  });

  detailBody.addEventListener("submit", async (event) => {
    if (event.target.id !== "feedbackForm") {
      return;
    }
    event.preventDefault();
    const button = event.target.querySelector("#feedbackSubmitBtn");
    if (button) {
      button.disabled = true;
    }
    try {
      if (typeof tinymce !== "undefined") {
        tinymce.triggerSave();
      }
      const formData = new FormData(event.target);
      formData.set("action", "save_feedback");
      formData.set("class_id", String(detailClassId));
      const response = await fetch(pageUrl, {
        method: "POST",
        body: formData,
      });
      const data = await response.json();
      if (data.status !== "ok") {
        throw new Error(data.message || "Could not save feedback.");
      }
      await loadSchedule();
      openDetailModal(detailClassId);
    } catch (error) {
      Swal.fire({
        icon: "error",
        title: "Could not save feedback.",
        text: error.message || "Could not save feedback.",
      });

      if (button) {
        button.disabled = false;
      }
    }
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const dateInput = document.getElementById("classDate");
    const startTimeInput = document.getElementById("classStartTime");
    const dateValue = dateInput?.value;
    const timeValue = startTimeInput?.value;

    if (isClassScheduleInPast(dateValue, timeValue)) {
      Swal.fire({
        icon: "error",
        title: "Schedule Error",
        text: "You cannot schedule a class in the past. Please select a future date and time.",
      });
      return;
    }
    submitBtn.disabled = true;
    try {
      const response = await fetch(pageUrl, {
        method: "POST",
        body: new FormData(form),
      });
      const data = await response.json();
      if (data.status !== "ok") {
        throw new Error(data.message || "Could not schedule the class.");
      }
      closeModal();
      await loadSchedule();
    } catch (error) {
      Swal.fire({
        icon: "error",
        title: "Schedule Error",
        text: error.message || "Could not schedule the class.",
      });

    } finally {
      submitBtn.disabled = enrollmentSelect.options.length <= 1;
    }
  });

  enrollmentSelect.addEventListener("change", updateClassPreview);
  classStartInput.addEventListener("input", updateClassPreview);
  classDateInput.addEventListener("change", () => {
    updateScheduleDateTimeLimits();
    updateClassPreview();
  });


  cellsEl.addEventListener("click", (event) => {
    const block = event.target.closest(".calendar-event");
    if (block) {
      event.stopPropagation();
      openDetailModal(Number(block.dataset.classId));
      return;
    }
    const cell = event.target.closest("[data-hour]");
    if (cell) {
      const hour = Number(cell.dataset.hour);
      const now = new Date();
      const selectedDate = startOfDay(state.date);
      const selectedDateTime = new Date(selectedDate);
      selectedDateTime.setHours(hour, 0, 0, 0);
      if (selectedDateTime <= now) {
        return;
      }
      openAddModal(hour);
    }
  });

  cellsEl.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    if (event.target.closest(".calendar-event")) {
      return;
    }

    const cell = event.target.closest("[data-hour]");
    if (!cell) {
      return;
    }
    event.preventDefault();
    const hour = Number(cell.dataset.hour);
    const now = new Date();
    const selectedDate = startOfDay(state.date);
    const selectedDateTime = new Date(selectedDate);
    selectedDateTime.setHours(hour, 0, 0, 0);
    if (selectedDateTime <= now) {
      return;
    }
    openAddModal(hour);
  });

  monthGrid.addEventListener("click", (event) => {
    const button = event.target.closest("[data-date]");
    if (!button) {
      return;
    }
    state.view = "day";
    state.date = parseLocalDate(button.dataset.date);
    applyViewButtonStyles();
    refreshDateControls();
    loadSchedule();
  });

  newClassButton.addEventListener("click", () => {
    openAddModal();
  });

  modal.addEventListener("click", (event) => {
    if (event.target.closest("[data-close-modal]")) {
      closeModal();
    }
  });

  detailModal.addEventListener("click", (event) => {
    if (event.target.closest("[data-close-detail]")) {
      closeDetailModal();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }
    if (!modal.classList.contains("hidden")) {
      closeModal();
    }
    if (!detailModal.classList.contains("hidden")) {
      closeDetailModal();
    }
  });

  viewDayBtn.addEventListener("click", () => setView("day"));
  viewMonthBtn.addEventListener("click", () => setView("month"));
  prevBtn.addEventListener("click", () => shiftDate(-1));
  todayBtn.addEventListener("click", () => goToDate(new Date()));
  nextBtn.addEventListener("click", () => shiftDate(1));
  dateInput.addEventListener("change", () => {
    if (!dateInput.value) {
      return;
    }
    goToDate(parseLocalDate(dateInput.value));
  });

  function isClassScheduleInPast(dateValue, timeValue) {
    if (!dateValue || !timeValue) {
      return false;
    }
    const selectedDateTime = new Date(`${dateValue}T${timeValue}`);
    const now = new Date();
    return selectedDateTime <= now;
  }
  
  function isClassScheduleInPast(dateValue, timeValue) {
    if (!dateValue || !timeValue) {
      return false;
    }
    const selectedDateTime = new Date(`${dateValue}T${timeValue}`);
    const now = new Date();
    if (Number.isNaN(selectedDateTime.getTime())) {
      return false;
    }
    return selectedDateTime <= now;
  }

  function setMinimumClassDate() {
    const dateInput = document.getElementById("classDate");
    if (!dateInput) {
      return;
    }
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, "0");
    const day = String(now.getDate()).padStart(2, "0");

    dateInput.min = `${year}-${month}-${day}`;
  }
  
  function updateScheduleDateTimeLimits() {
    if (!classDateInput || !classStartInput) {
      return;
    }
    const now = new Date();
    // Minimum selectable date = today.
    classDateInput.min = toDateString(now);
    // If scheduling for today, prevent times that have already passed.
    if (classDateInput.value === toDateString(now)) {
      classStartInput.min = `${pad2(now.getHours())}:${pad2(now.getMinutes())}`;
    } else {
      // Future dates can use any valid time.
      classStartInput.removeAttribute("min");
    }
  }

  /* =====================================================
    INITIALIZE
  ====================================================== */

  buildGridSkeleton();
  applyViewButtonStyles();
  refreshDateControls();
  updateScheduleDateTimeLimits();
  loadSchedule();
})();
