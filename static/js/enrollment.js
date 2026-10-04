function toggleStudentMode() {
  const selected = document.querySelector('input[name="student_type"]:checked');

  if (!selected) {
    return;
  }

  const isNew = selected.value === "new";
  const existingSection = document.getElementById("existing-student-section");
  const newSection = document.getElementById("new-student-section");
  const studentSelect = document.getElementById("StudentID");

  if (!existingSection || !newSection || !studentSelect) {
    return;
  }

  existingSection.classList.toggle("hidden", isNew);
  newSection.classList.toggle("hidden", !isNew);
  newSection.classList.toggle("grid", isNew);

  studentSelect.required = !isNew;

  newSection.querySelectorAll("input").forEach((input) => {
    input.required = isNew;
  });

  document.querySelectorAll(".student-type-card").forEach((card) => {
    const radio = card.querySelector('input[type="radio"]');

    if (!radio) {
      return;
    }

    if (radio.checked) {
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

  if (!selectedOption || !selectedOption.value) {
    totalClasses.value = "";
    summary?.classList.add("hidden");
    return;
  }

  const numberOfClasses = Number(selectedOption.dataset.numberOfClasses || 0);

  const numberOfFreeClasses = Number(
    selectedOption.dataset.numberOfFreeClasses || 0,
  );

  const total = numberOfClasses + numberOfFreeClasses;

  const price = selectedOption.dataset.price || "";
  const validFrom = selectedOption.dataset.validFrom || "";
  const validUntil = selectedOption.dataset.validUntil || "";

  if (!Number.isFinite(total) || total <= 0) {
    totalClasses.value = "";
    summary?.classList.add("hidden");
    return;
  }

  const classLabel = total === 1 ? "class" : "classes";

  totalClasses.value = total;

  if (summary && summaryName && summaryText) {
    summaryName.textContent = selectedOption.textContent.trim();

    const details = [`${total} ${classLabel} included`];

    if (numberOfFreeClasses > 0) {
      details.push(`${numberOfFreeClasses} free`);
    }

    if (price) {
      details.push(`Price: ${price}`);
    }

    if (validFrom && validUntil) {
      details.push(
        `Valid ${formatPackageDate(validFrom)} – ${formatPackageDate(validUntil)}`,
      );
    }

    summaryText.textContent = details.join(" · ");
    summary.classList.remove("hidden");

    refreshIcons();
  }
}

function formatPackageDate(value) {
  if (!value) {
    return "";
  }

  const date = new Date(`${value}T00:00:00`);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
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

  if (!panel || !reference || !credsBox || !creds) {
    return;
  }

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

  if (!errorBox) {
    return;
  }

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

  if (!errorBox) {
    return;
  }

  errorBox.textContent = "";
  errorBox.classList.add("hidden");
}

function setSubmitLoading(loading) {
  const button = document.getElementById("enrollmentSubmitBtn");

  if (!button) {
    return;
  }

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

function validateEnrollmentForm(formData) {
  const studentType = formData.get("student_type");
  const courseID = String(formData.get("CourseID") || "").trim();
  const packageID = String(formData.get("PackageID") || "").trim();

  if (!studentType) {
    return "Select whether this is an existing or new student.";
  }

  if (studentType === "existing") {
    const studentID = String(formData.get("StudentID") || "").trim();

    if (!studentID) {
      return "Select a student.";
    }
  }

  if (studentType === "new") {
    const firstName = String(formData.get("NewStudentFirstName") || "").trim();

    const lastName = String(formData.get("NewStudentLastName") || "").trim();

    const weChat = String(formData.get("NewStudentWeChat") || "").trim();

    const email = String(formData.get("NewStudentEmail") || "").trim();

    if (!firstName || !lastName) {
      return "First name and last name are required.";
    }

    if (!weChat) {
      return "WeChat ID is required.";
    }

    if (!email) {
      return "Enter a valid email address.";
    }
  }

  if (!courseID) {
    return "Select a course.";
  }

  if (!packageID) {
    return "Select a package.";
  }

const numberOfClasses = Number(selectedPackage?.dataset.numberOfClasses || 0);
const numberOfFreeClasses = Number(
  selectedPackage?.dataset.numberOfFreeClasses || 0,
);

const total = numberOfClasses + numberOfFreeClasses;

if (!Number.isFinite(total) || total <= 0) {
  return "The selected package has no available classes.";
}

  return "";
}

async function submitEnrollment(event) {
  event.preventDefault();

  const form = event.currentTarget;

  hideEnrollmentError();

  const formData = new FormData(form);
  const validationError = validateEnrollmentForm(formData);

  if (validationError) {
    showEnrollmentError(validationError);
    return;
  }

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

      if (studentSelect) {
        studentSelect.add(new Option(name, data.student_id));
      }
    }

    form.reset();

    toggleStudentMode();
    updateTotalClasses();

    const contractFileName = document.getElementById("contractFileName");

    if (contractFileName) {
      contractFileName.textContent = "";
      contractFileName.classList.add("hidden");
    }

    showEnrollmentResult(data);

    await reloadEnrollmentTable();
  } catch (error) {
    console.error("Error saving enrollment:", error);
    showEnrollmentError(error.message || "Could not create the enrollment.");
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

function applyEnrollmentFilters() {
  const searchInput = document.getElementById("enrollmentSearch");

  const studentFilter = document.getElementById("historyStudentFilter");

  const empty = document.getElementById("enrollmentSearchEmpty");

  const rows = Array.from(document.querySelectorAll(".enrollment-row"));

  if (!searchInput || !studentFilter || !empty) {
    return;
  }

  const query = searchInput.value.trim().toLowerCase();

  const selectedStudentID = studentFilter.value;

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

  if (clearButton) {
    clearButton.classList.toggle("hidden", !searchInput.value);
  }

  updateSelectedStudentBanner();
}

function updateSelectedStudentBanner() {
  const studentFilter = document.getElementById("historyStudentFilter");

  const banner = document.getElementById("selectedStudentHistory");

  const name = document.getElementById("selectedStudentHistoryName");

  if (!studentFilter || !banner || !name) {
    return;
  }

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

  if (!input || !clearButton) {
    return;
  }

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

  if (!studentFilter || !clearButton) {
    return;
  }

  studentFilter.addEventListener("change", applyEnrollmentFilters);

  clearButton.addEventListener("click", () => {
    studentFilter.value = "";
    applyEnrollmentFilters();
  });
}

function setupContractInput() {
  const input = document.getElementById("Contract");

  const fileName = document.getElementById("contractFileName");

  if (!input || !fileName) {
    return;
  }

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

  if (!button) {
    return;
  }

  button.addEventListener("click", async () => {
    const creds = document.getElementById("enrollmentCreds");

    const text = creds?.textContent || "";

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

  const form = document.getElementById("enrollmentForm");

  if (form) {
    form.addEventListener("submit", submitEnrollment);
  }

  setupEnrollmentSearch();
  setupStudentHistoryFilter();
  setupContractInput();
  setupCredentialCopy();
  applyEnrollmentFilters();
  refreshIcons();
});
