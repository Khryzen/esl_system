(() => {
  "use strict";

  const pageUrl = window.location.pathname;

  function escapeHtml(value) {
    const div = document.createElement("div");
    div.textContent = value ?? "";
    return div.innerHTML;
  }

  async function refreshDashboardSummary() {
    try {
      const body = new URLSearchParams({
        action: "refresh_dashboard",
      });

      const response = await fetch(pageUrl, {
        method: "POST",
        body,
      });

      const contentType = response.headers.get("content-type") || "";

      if (!contentType.includes("application/json")) {
        throw new Error(`Server returned ${response.status}.`);
      }

      const data = await response.json();

      if (data.status !== "ok") {
        throw new Error(data.message || "Could not refresh the dashboard.");
      }

      updateStatCards(data);
      updateRenewals(data);
      updateAttendanceFollowUps(data);
    } catch (error) {
      console.error("Error refreshing dashboard:", error);

      if (typeof Swal !== "undefined") {
        Swal.fire({
          icon: "error",
          title: "Dashboard Error",
          text: error.message || "Could not refresh the dashboard.",
        });
      }
    }
  }

  function updateStatCards(data) {
    const studentCountEl = document.getElementById("dashboardStudentCount");

    const courseCountEl = document.getElementById("dashboardCourseCount");

    const packageCountEl = document.getElementById("dashboardPackageCount");

    if (studentCountEl) {
      studentCountEl.textContent = data.number_of_students ?? 0;
    }

    if (courseCountEl) {
      courseCountEl.textContent = data.number_of_courses ?? 0;
    }

    if (packageCountEl) {
      packageCountEl.textContent = data.number_of_packages ?? 0;
    }
  }

  function updateRenewals(data) {
    const totalEl = document.getElementById("renewalsTotal");

    const listEl = document.getElementById("renewalsList");

    const moreEl = document.getElementById("renewalsMore");

    const renewals = Array.isArray(data.renewals) ? data.renewals : [];

    const total = Number(data.renewals_total || 0);

    const more = Number(data.renewals_more || 0);

    if (totalEl) {
      totalEl.textContent = total;
      totalEl.classList.toggle("hidden", total <= 0);
    }

    if (listEl) {
      if (renewals.length === 0) {
        listEl.innerHTML = `
          <div class="px-5 py-8 text-center">
            <p class="text-sm font-medium text-slate-700 dark:text-slate-300">
              No renewals needed
            </p>

            <p class="mt-0.5 text-xs text-slate-400 dark:text-slate-500">
              Everyone has sufficient classes.
            </p>
          </div>
        `;
      } else {
        listEl.innerHTML = renewals
          .map(
            (item) => `
              <div class="action-row px-5 py-3.5">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-slate-900 dark:text-white">
                      ${escapeHtml(item.Student || "—")}
                    </p>

                    <p class="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
                      ${escapeHtml(item.Course || "—")}
                      ·
                      ${escapeHtml(item.Package || "—")}
                    </p>
                  </div>

                  <span class="shrink-0 rounded-full bg-amber-50 px-2 py-1 text-[11px] font-semibold text-amber-700 dark:bg-amber-950/50 dark:text-amber-300">
                    ${Number(item.ClassesRemaining || 0)} left
                  </span>
                </div>
              </div>
            `,
          )
          .join("");
      }
    }

    if (moreEl) {
      moreEl.textContent = more > 0 ? `+${more} more` : "";

      moreEl.classList.toggle("hidden", more <= 0);
    }

    if (window.lucide) {
      window.lucide.createIcons();
    }
  }

  function updateAttendanceFollowUps(data) {
    const totalEl = document.getElementById("attendanceFollowUpsTotal");

    const listEl = document.getElementById("followUpList");

    const moreEl = document.getElementById("attendanceFollowUpsMore");

    const followUps = Array.isArray(data.attendance_follow_ups)
      ? data.attendance_follow_ups
      : [];

    const total = Number(data.attendance_follow_ups_total || 0);

    const more = Number(data.attendance_follow_ups_more || 0);

    if (totalEl) {
      totalEl.textContent = total;

      totalEl.classList.toggle("hidden", total <= 0);
    }

    if (listEl) {
      if (followUps.length === 0) {
        listEl.innerHTML = `
          <div class="px-5 py-8 text-center">
            <p class="text-sm font-medium text-slate-700 dark:text-slate-300">
              All caught up
            </p>

            <p class="mt-0.5 text-xs text-slate-400 dark:text-slate-500">
              No attendance requires follow-up.
            </p>
          </div>
        `;
      } else {
        listEl.innerHTML = followUps
          .map(
            (item) => `
              <a
                href="/class/?date=${encodeURIComponent(
                  item.DateISO || "",
                )}&class_id=${Number(item.ID || 0)}"
                class="action-row block px-5 py-3.5"
              >
                <div class="flex items-start gap-3">
                  <div class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400">
                    <i data-lucide="user" class="h-3.5 w-3.5"></i>
                  </div>

                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-slate-900 dark:text-white">
                      ${escapeHtml(item.Student || "—")}
                    </p>

                    <p class="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
                      ${escapeHtml(item.Course || "—")}
                    </p>

                    <p class="mt-1 text-[11px] text-slate-400 dark:text-slate-500">
                      ${escapeHtml(item.Date || "—")}
                      ·
                      ${escapeHtml(item.Time || "—")}
                    </p>
                  </div>

                  <i
                    data-lucide="chevron-right"
                    class="ml-auto mt-1 h-4 w-4 shrink-0 text-slate-300 dark:text-slate-600"
                  ></i>
                </div>
              </a>
            `,
          )
          .join("");
      }
    }

    if (moreEl) {
      moreEl.textContent = more > 0 ? `+${more} more` : "";
      moreEl.classList.toggle("hidden", more <= 0);
    }
    if (window.lucide) {
      window.lucide.createIcons();
    }
  }
  window.refreshDashboardSummary = refreshDashboardSummary;
})();
