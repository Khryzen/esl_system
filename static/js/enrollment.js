function toggleStudentMode() {
  const selected = document.querySelector('input[name="student_type"]:checked');

  if (!selected) {
    return;
  }

  const isNew = selected.value === "new";

  const existingSection = document.getElementById("existing-student-section");

  const newSection = document.getElementById("new-student-section");

  const studentSelect = document.getElementById("StudentID");

  existingSection.classList.toggle("hidden", isNew);

  newSection.classList.toggle("hidden", !isNew);

  newSection.classList.toggle("grid", isNew);

  studentSelect.required = !isNew;

  newSection.querySelectorAll("input").forEach((input) => {
    input.required = isNew;
  });

  document.querySelectorAll(".student-type-card").forEach((card) => {
    const radio = card.querySelector('input[type="radio"]');

    if (radio?.checked) {
      card.classList.add(
        "border-blue-200",
        "bg-blue-50/70",
        "dark:border-blue-500/30",
        "dark:bg-blue-500/5",
      );

      card.classList.remove(
        "border-slate-200",
        "bg-white",
        "dark:border-slate-700",
        "dark:bg-slate-900",
      );
    } else {
      card.classList.remove(
        "border-blue-200",
        "bg-blue-50/70",
        "dark:border-blue-500/30",
        "dark:bg-blue-500/5",
      );

      card.classList.add(
        "border-slate-200",
        "bg-white",
        "dark:border-slate-700",
        "dark:bg-slate-900",
      );
    }
  });
}

function updateTotalClasses() {
  const packageSelect = document.getElementById("PackageID");

  const totalClasses = document.getElementById("TotalClasses");

  const summary = document.getElementById("packageSummary");

  const summaryName = document.getElementById("packageSummaryName");

  const summaryText = document.getElementById("packageSummaryText");

  if (!packageSelect || !totalClasses) {
    return;
  }

  const selectedOption = packageSelect.options[packageSelect.selectedIndex];

  if (
    !selectedOption ||
    !selectedOption.value ||
    !selectedOption.dataset.total
  ) {
    totalClasses.value = "";

    summary?.classList.add("hidden");

    return;
  }

  const total = selectedOption.dataset.total;

  totalClasses.value = total;

  if (summary && summaryName && summaryText) {
    summaryName.textContent = selectedOption.textContent.trim();

    summaryText.textContent = `${total} class${Number(total) === 1 ? "" : "es"} will be added to this enrollment.`;

    summary.classList.remove("hidden");
  }

  refreshIcons();
}

function refreshIcons() {
  if (typeof lucide !== "undefined") {
    lucide.createIcons();
  }
}

function credsToText(creds) {
  if (!creds || typeof creds !== "object") {
    return String(creds || "");
  }

  return Object.entries(creds)
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n");
}

function showEnrollmentResult(data) {
  const panel = document.getElementById("enrollmentResult");

  const reference = document.getElementById("enrollmentRef");

  const credsBox = document.getElementById("enrollmentCredsBox");

  const creds = document.getElementById("enrollmentCreds");

  reference.textContent = data.reference_number || "";

  if (data.creds) {
    creds.textContent = credsToText(data.creds);

    credsBox.classList.remove("hidden");
  } else {
    creds.textContent = "";

    credsBox.classList.add("hidden");
  }

  panel.classList.remove("hidden");

  refreshIcons();

  panel.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}

function showEnrollmentError(message) {
  const errorBox = document.getElementById("enrollmentError");

  errorBox.textContent =
    message || "Something went wrong while saving the enrollment.";

  errorBox.classList.remove("hidden");

  errorBox.scrollIntoView({
    behavior: "smooth",
    block: "center",
  });
}

function hideEnrollmentError() {
  const errorBox = document.getElementById("enrollmentError");

  errorBox.textContent = "";

  errorBox.classList.add("hidden");
}

function setSubmitLoading(loading) {
  const button = document.getElementById("enrollmentSubmitBtn");

  button.disabled = loading;

  if (loading) {
    button.innerHTML = `
      <i data-lucide="loader-circle" class="h-4 w-4 animate-spin"></i>
      Creating...
    `;
  } else {
    button.innerHTML = `
      <i data-lucide="clipboard-check" class="h-4 w-4"></i>
      Create Enrollment
    `;
  }

  refreshIcons();
}

async function submitEnrollment(event) {
  event.preventDefault();

  const form = event.currentTarget;

  hideEnrollmentError();

  const formData = new FormData(form);

  const isNewStudent = formData.get("student_type") === "new";

  const file = formData.get("Contract");

  if (file instanceof File && file.size === 0) {
    formData.delete("Contract");
  }

  setSubmitLoading(true);

  try {
    const response = await fetch(window.location.pathname, {
      method: "POST",
      body: formData,
    });

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (data.status !== "ok") {
      throw new Error(data.message || "Could not create the enrollment.");
    }

    if (isNewStudent && data.student_id) {
      const firstName = formData.get("NewStudentFirstName") || "";

      const lastName = formData.get("NewStudentLastName") || "";

      const name = `${firstName} ${lastName}`.trim();

      const studentSelect = document.getElementById("StudentID");

      studentSelect.add(new Option(name, data.student_id));
    }

    form.reset();

    toggleStudentMode();
    updateTotalClasses();

    const contractFileName = document.getElementById("contractFileName");

    contractFileName.textContent = "";

    contractFileName.classList.add("hidden");

    showEnrollmentResult(data);

    await reloadEnrollmentTable();
  } catch (error) {
    console.error("Error saving enrollment:", error);

    showEnrollmentError(error.message);
  } finally {
    setSubmitLoading(false);
  }
}

async function reloadEnrollmentTable() {
  try {
    const response = await fetch(window.location.pathname, {
      method: "GET",
      headers: {
        Accept: "text/html",
      },
    });

    if (!response.ok) {
      throw new Error(`Could not refresh enrollments (${response.status}).`);
    }

    const html = await response.text();

    const parser = new DOMParser();

    const parsed = parser.parseFromString(html, "text/html");

    const newTable = parsed.getElementById("coursesTable");

    const currentTable = document.getElementById("coursesTable");

    if (!newTable || !currentTable) {
      return;
    }

    currentTable.innerHTML = newTable.innerHTML;

    applyEnrollmentFilters();

    refreshIcons();
  } catch (error) {
    console.error("Failed to refresh enrollment table:", error);
  }
}

/*
 * Applies BOTH filters:
 *
 * 1. Selected student
 * 2. Search text
 *
 * This is the central filtering function so the two
 * controls cannot accidentally override each other.
 */
function applyEnrollmentFilters() {
  const searchInput = document.getElementById("enrollmentSearch");

  const studentFilter = document.getElementById("historyStudentFilter");

  const empty = document.getElementById("enrollmentSearchEmpty");

  const rows = Array.from(document.querySelectorAll(".enrollment-row"));

  const query = searchInput.value.trim().toLowerCase();

  const selectedStudentID = studentFilter.value;

  /*
   * Build the selected student's name from
   * the student dropdown.
   */
  let selectedStudentName = "";

  if (selectedStudentID) {
    const option = studentFilter.options[studentFilter.selectedIndex];

    selectedStudentName = option?.textContent.trim().toLowerCase() || "";
  }

  let matches = 0;

  rows.forEach((row) => {
    const rowStudent = (row.dataset.student || "").trim().toLowerCase();

    const searchText = (
      row.dataset.search ||
      row.textContent ||
      ""
    ).toLowerCase();

    /*
     * Match by the student name shown in the
     * history table.
     */
    const matchesStudent =
      !selectedStudentName || rowStudent === selectedStudentName;

    const matchesSearch = !query || searchText.includes(query);

    const visible = matchesStudent && matchesSearch;

    row.classList.toggle("hidden", !visible);

    if (visible) {
      matches++;
    }
  });

  empty.classList.toggle("hidden", matches !== 0 || rows.length === 0);

  const clearButton = document.getElementById("clearEnrollmentSearch");

  clearButton.classList.toggle("hidden", !searchInput.value);

  updateSelectedStudentBanner();
}

function updateSelectedStudentBanner() {
  const studentFilter = document.getElementById("historyStudentFilter");

  const banner = document.getElementById("selectedStudentHistory");

  const name = document.getElementById("selectedStudentHistoryName");

  if (!studentFilter.value) {
    banner.classList.add("hidden");

    name.textContent = "";

    return;
  }

  const option = studentFilter.options[studentFilter.selectedIndex];

  name.textContent = option?.textContent.trim() || "Selected student";

  banner.classList.remove("hidden");

  refreshIcons();
}

function setupEnrollmentSearch() {
  const input = document.getElementById("enrollmentSearch");

  const clearButton = document.getElementById("clearEnrollmentSearch");

  input.addEventListener("input", applyEnrollmentFilters);

  clearButton.addEventListener("click", () => {
    input.value = "";

    applyEnrollmentFilters();

    input.focus();
  });
}

function setupStudentHistoryFilter() {
  const studentFilter = document.getElementById("historyStudentFilter");

  const clearButton = document.getElementById("clearStudentHistory");

  studentFilter.addEventListener("change", () => {
    applyEnrollmentFilters();
  });

  clearButton.addEventListener("click", () => {
    studentFilter.value = "";

    applyEnrollmentFilters();
  });
}

function setupContractInput() {
  const input = document.getElementById("Contract");

  const fileName = document.getElementById("contractFileName");

  input.addEventListener("change", () => {
    const file = input.files?.[0];

    if (!file) {
      fileName.textContent = "";

      fileName.classList.add("hidden");

      return;
    }

    fileName.textContent = `${file.name} · ${formatFileSize(file.size)}`;

    fileName.classList.remove("hidden");
  });
}

function formatFileSize(bytes) {
  if (bytes < 1024) {
    return `${bytes} B`;
  }

  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }

  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function setupCredentialCopy() {
  const button = document.getElementById("copyEnrollmentCreds");

  button.addEventListener("click", async () => {
    const text = document.getElementById("enrollmentCreds").textContent;

    if (!text) {
      return;
    }

    try {
      await navigator.clipboard.writeText(text);

      const original = button.innerHTML;

      button.innerHTML = `
          <i data-lucide="check" class="h-3.5 w-3.5"></i>
          Copied
        `;

      refreshIcons();

      setTimeout(() => {
        button.innerHTML = original;

        refreshIcons();
      }, 1200);
    } catch (error) {
      console.error("Could not copy credentials:", error);
    }
  });
}

document.addEventListener("DOMContentLoaded", () => {
  toggleStudentMode();

  updateTotalClasses();

  document
    .getElementById("enrollmentForm")
    .addEventListener("submit", submitEnrollment);

  setupEnrollmentSearch();

  setupStudentHistoryFilter();

  setupContractInput();

  setupCredentialCopy();

  applyEnrollmentFilters();

  refreshIcons();
});
