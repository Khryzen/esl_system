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
      card.classList.add("border-blue-200", "bg-blue-50/70", "dark:border-blue-500/30", "dark:bg-blue-500/5");

      card.classList.remove("border-slate-200", "bg-white", "dark:border-slate-700", "dark:bg-slate-900");
    } else {
      card.classList.remove("border-blue-200", "bg-blue-50/70", "dark:border-blue-500/30", "dark:bg-blue-500/5");

      card.classList.add("border-slate-200", "bg-white", "dark:border-slate-700", "dark:bg-slate-900");
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

  const numberOfFreeClasses = Number(selectedOption.dataset.numberOfFreeClasses || 0);

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
      details.push(`Valid ${formatPackageDate(validFrom)} – ${formatPackageDate(validUntil)}`);
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

  errorBox.textContent = message || "Something went wrong while saving the enrollment.";

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

  const packageSelect = document.getElementById("PackageID");
  const selectedPackage = packageSelect?.options[packageSelect.selectedIndex];

  const numberOfClasses = Number(selectedPackage?.dataset.numberOfClasses || 0);

  const numberOfFreeClasses = Number(selectedPackage?.dataset.numberOfFreeClasses || 0);

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

    const searchText = (row.dataset.search || row.textContent || "").toLowerCase();

    const matchesStudent = !selectedStudentName || rowStudent === selectedStudentName;

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

let currentEnrollmentDetailsID = null;
let currentEnrollmentDetailsActive = false;
let currentEnrollmentDetailsRemaining = 0;
let currentRenewalStudentName = "";

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
  setupEnrollmentDetails();
});

function formatDetailsDate(value) {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatDetailsAmount(value) {
  const amount = Number(value);

  if (!Number.isFinite(amount)) {
    return "—";
  }

  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);
}

function setEnrollmentDetailsText(id, value) {
  const element = document.getElementById(id);

  if (element) {
    element.textContent = value ?? "—";
  }
}

function openEnrollmentDetailsModal() {
  const modal = document.getElementById("enrollmentDetailsModal");

  if (!modal) {
    return;
  }

  modal.classList.remove("hidden");
  modal.setAttribute("aria-hidden", "false");
  document.body.classList.add("overflow-hidden");
}

function closeEnrollmentDetailsModal() {
  const modal = document.getElementById("enrollmentDetailsModal");

  if (!modal) {
    return;
  }

  modal.classList.add("hidden");
  modal.setAttribute("aria-hidden", "true");
  document.body.classList.remove("overflow-hidden");
}

function resetEnrollmentDetailsModal() {
  setEnrollmentDetailsText("enrollmentDetailsReference", "—");
  setEnrollmentDetailsText("detailsStudent", "—");
  setEnrollmentDetailsText("detailsStatus", "—");
  setEnrollmentDetailsText("detailsCourse", "—");
  setEnrollmentDetailsText("detailsPackage", "—");
  setEnrollmentDetailsText("detailsTotalClasses", "—");
  setEnrollmentDetailsText("detailsClassesRemaining", "—");
  

  currentEnrollmentDetailsID = null;
  currentEnrollmentDetailsActive = false;
  currentEnrollmentDetailsRemaining = 0;

  const changeCourseSection = document.getElementById("changeCourseSection");
  const changeCourseSelect = document.getElementById("detailsChangeCourse");
  const changeCourseButton = document.getElementById("changeEnrollmentCourse");
  const changeCourseUnavailable = document.getElementById("changeCourseUnavailable");

  if (changeCourseSection) {
    changeCourseSection.classList.add("hidden");
  }

  if (changeCourseSelect) {
    changeCourseSelect.value = "";
    changeCourseSelect.disabled = true;
  }

  if (changeCourseButton) {
    changeCourseButton.disabled = true;
  }

  if (changeCourseUnavailable) {
    changeCourseUnavailable.classList.add("hidden");
  }

  const contractLink = document.getElementById("detailsContractLink");
  const contractStatus = document.getElementById("detailsContractStatus");

  if (contractLink) {
    contractLink.href = "#";
    contractLink.classList.add("hidden");
  }

  if (contractStatus) {
    contractStatus.textContent = "None";
  }

  const invoiceContent = document.getElementById("detailsInvoiceContent");
  const invoiceStatus = document.getElementById("detailsInvoiceStatus");
  const viewInvoiceButton = document.getElementById("viewEnrollmentInvoice");

  if (invoiceContent) {
    invoiceContent.classList.add("hidden");
  }

  if (invoiceStatus) {
    invoiceStatus.textContent = "No invoice";
  }

  if (viewInvoiceButton) {
    viewInvoiceButton.classList.add("hidden");
    viewInvoiceButton.disabled = true;
  }

  setEnrollmentDetailsText("detailsInvoiceNumber", "—");
  setEnrollmentDetailsText("detailsInvoiceAmount", "—");
  setEnrollmentDetailsText("detailsInvoiceDate", "—");
  setEnrollmentDetailsText("detailsInvoiceDueDate", "—");
  setEnrollmentDetailsText("detailsInvoiceTransaction", "—");
  
  document.getElementById("detailsHistory").innerHTML = "";
  document.getElementById("detailsClassHistory").innerHTML = "";

  const classHistory = document.getElementById("detailsClassHistory");

  if (classHistory) {
    classHistory.innerHTML = "";
  }

  const history = document.getElementById("detailsHistory");

  if (history) {
    history.innerHTML = "";
  }
}

function setupChangeCourseState(enrollment) {
  const section = document.getElementById("changeCourseSection");

  const select = document.getElementById("detailsChangeCourse");

  const button = document.getElementById("changeEnrollmentCourse");

  const unavailable = document.getElementById("changeCourseUnavailable");

  if (!section || !select || !button || !unavailable) {
    return;
  }

  section.classList.remove("hidden");
  select.value = "";

  Array.from(select.options).forEach((option) => {
    if (!option.value) {
      return;
    }

    option.disabled = String(option.value) === String(enrollment.course_id);
  });

  const canChange = Boolean(enrollment.active) && Number(enrollment.classes_remaining || 0) > 0;

  select.disabled = !canChange;
  button.disabled = true;

  unavailable.classList.toggle("hidden", canChange);

  if (!canChange) {
    unavailable.textContent = enrollment.active
      ? "Course changes are unavailable because this enrollment has no classes remaining."
      : "Course changes are unavailable because this enrollment is inactive.";
  }
}

function setupChangeCourseHandler() {
  const select = document.getElementById("detailsChangeCourse");

  const button = document.getElementById("changeEnrollmentCourse");

  if (!select || !button) {
    return;
  }

  select.addEventListener("change", () => {
    button.disabled =
      !select.value ||
      !currentEnrollmentDetailsID ||
      !currentEnrollmentDetailsActive ||
      currentEnrollmentDetailsRemaining <= 0;
  });

  button.addEventListener("click", changeEnrollmentCourse);
}

async function changeEnrollmentCourse() {
  const select = document.getElementById("detailsChangeCourse");

  const button = document.getElementById("changeEnrollmentCourse");

  if (!select || !button) {
    return;
  }

  const enrollmentID = currentEnrollmentDetailsID;
  const courseID = select.value;

  if (!enrollmentID || !courseID) {
    return;
  }

  const courseName = select.options[select.selectedIndex]?.textContent.trim() || "the selected course";

  const confirmation = await Swal.fire({
    icon: "question",
    title: "Change course?",
    text: `Move the remaining classes to ${courseName}?`,
    showCancelButton: true,
    confirmButtonText: "Change Course",
    cancelButtonText: "Cancel",
    reverseButtons: true,
  });

  if (!confirmation.isConfirmed) {
    return;
  }

  button.disabled = true;

  try {
    const formData = new URLSearchParams();

    formData.set("enrollment_id", String(enrollmentID));

    formData.set("course_id", String(courseID));

    const response = await fetch("/enrollment/change-course/", {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Accept: "application/json",
      },
      body: formData.toString(),
    });

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (!response.ok || data.status !== "ok") {
      throw new Error(data.message || "Could not change the enrollment course.");
    }

    await reloadEnrollmentTable();
    await showEnrollmentDetails(enrollmentID);

    await Swal.fire({
      icon: "success",
      title: "Course changed",
      text: "The enrollment course was updated successfully.",
      timer: 1600,
      showConfirmButton: false,
    });
  } catch (error) {
    console.error("Error changing enrollment course:", error);

    button.disabled = false;

    await Swal.fire({
      icon: "error",
      title: "Could not change course",
      text: error.message || "Could not change the enrollment course.",
    });
  }
}

async function viewEnrollmentInvoice() {
  if (!currentEnrollmentDetailsID) {
    return;
  }

  try {
    const response = await fetch(
      `/enrollment/invoice/?enrollment_id=${encodeURIComponent(currentEnrollmentDetailsID)}`,
      {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
      },
    );

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (!response.ok || data.status !== "ok" || !data.invoice) {
      throw new Error(data.message || "Could not load the enrollment invoice.");
    }

    const invoice = data.invoice;

    const paidText = invoice.paid ? "Paid" : "Unpaid";

    const transactionText = invoice.transaction_id ? escapeHtml(invoice.transaction_id) : "—";

    const paidDateText = invoice.paid_date ? formatDetailsDate(invoice.paid_date) : "—";

    await Swal.fire({
      icon: "info",
      title: escapeHtml(invoice.invoice_number || "Invoice"),
      html: `
        <div class="text-left">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <p class="text-xs text-slate-400">Status</p>
              <p class="mt-1 font-semibold">${paidText}</p>
            </div>

            <div>
              <p class="text-xs text-slate-400">Amount</p>
              <p class="mt-1 font-semibold">
                ${formatDetailsAmount(invoice.amount)}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-400">Invoice Date</p>
              <p class="mt-1 font-semibold">
                ${formatDetailsDate(invoice.invoice_date)}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-400">Due Date</p>
              <p class="mt-1 font-semibold">
                ${formatDetailsDate(invoice.due_date)}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-400">Paid Date</p>
              <p class="mt-1 font-semibold">
                ${paidDateText}
              </p>
            </div>

            <div>
              <p class="text-xs text-slate-400">Transaction ID</p>
              <p class="mt-1 break-all font-semibold">
                ${transactionText}
              </p>
            </div>
          </div>
        </div>
      `,
      confirmButtonText: "Close",
    });
  } catch (error) {
    console.error("Error loading enrollment invoice:", error);

    await Swal.fire({
      icon: "error",
      title: "Could not load invoice",
      text: error.message || "Could not load the enrollment invoice.",
    });
  }
}

async function showEnrollmentDetails(enrollmentID) {
  const modal = document.getElementById("enrollmentDetailsModal");

  const loading = document.getElementById("enrollmentDetailsLoading");

  const content = document.getElementById("enrollmentDetailsContent");

  if (!modal || !loading || !content) {
    return;
  }

  resetEnrollmentDetailsModal();

  loading.classList.remove("hidden");
  content.classList.add("hidden");

  openEnrollmentDetailsModal();

  try {
    const response = await fetch(`/enrollment/details/?id=${encodeURIComponent(enrollmentID)}`, {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    });

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (!response.ok || data.status !== "ok") {
      throw new Error(data.message || "Could not load enrollment details.");
    }

    const enrollment = data.enrollment;

    currentEnrollmentDetailsID = enrollment.id;
    currentEnrollmentDetailsActive = Boolean(enrollment.active);

    currentEnrollmentDetailsRemaining = Number(enrollment.classes_remaining || 0);
    currentRenewalStudentName = enrollment.student || "";
    setEnrollmentDetailsText("enrollmentDetailsReference", enrollment.reference_number);
    setEnrollmentDetailsText("detailsStudent", enrollment.student);
    setEnrollmentDetailsText("detailsStatus", enrollment.active ? "Active" : "Inactive");
    setEnrollmentDetailsText("detailsCourse", enrollment.course);
    setEnrollmentDetailsText("detailsPackage", enrollment.package);
    setEnrollmentDetailsText("detailsTotalClasses", enrollment.total_classes);
    setEnrollmentDetailsText("detailsClassesRemaining", enrollment.classes_remaining);

    setupChangeCourseState(enrollment);
    setupDeactivateEnrollmentState(enrollment);
    setupRenewEnrollmentState(enrollment);

    const contractLink = document.getElementById("detailsContractLink");

    const contractStatus = document.getElementById("detailsContractStatus");

    if (data.contract?.url) {
      if (contractLink) {
        contractLink.href = data.contract.url;
        contractLink.classList.remove("hidden");
      }

      if (contractStatus) {
        contractStatus.textContent = "Available";
      }
    } else if (contractStatus) {
      contractStatus.textContent = "None";
    }

    const invoiceContent = document.getElementById("detailsInvoiceContent");

    const invoiceStatus = document.getElementById("detailsInvoiceStatus");

    if (data.invoice) {
      if (invoiceContent) {
        invoiceContent.classList.remove("hidden");
      }

      if (invoiceStatus) {
        invoiceStatus.textContent = data.invoice.paid ? "Paid" : "Unpaid";
      }

      const viewInvoiceButton = document.getElementById("viewEnrollmentInvoice");

      if (viewInvoiceButton) {
        viewInvoiceButton.classList.remove("hidden");
        viewInvoiceButton.disabled = false;
      }

      setEnrollmentDetailsText("detailsInvoiceNumber", data.invoice.invoice_number);

      setEnrollmentDetailsText("detailsInvoiceAmount", formatDetailsAmount(data.invoice.amount));

      setEnrollmentDetailsText("detailsInvoiceDate", formatDetailsDate(data.invoice.invoice_date));

      setEnrollmentDetailsText("detailsInvoiceDueDate", formatDetailsDate(data.invoice.due_date));
      setEnrollmentDetailsText("detailsInvoiceTransaction", data.invoice.transaction_id || "—");
    }

    renderEnrollmentHistory(data.history || []);
    renderEnrollmentClassHistory(data.classes || []);

    loading.classList.add("hidden");
    content.classList.remove("hidden");

    refreshIcons();
  } catch (error) {
    console.error("Error loading enrollment details:", error);

    loading.classList.remove("hidden");
    content.classList.add("hidden");

    await Swal.fire({
      icon: "error",
      title: "Could not load enrollment",
      text: error.message || "Could not load enrollment details.",
    });

    closeEnrollmentDetailsModal();
  }
}

function renderEnrollmentHistory(history) {
  const container = document.getElementById("detailsHistory");

  if (!container) {
    return;
  }

  if (!history.length) {
    container.innerHTML = `
      <div class="rounded-xl border border-dashed border-slate-200 px-4 py-6 text-center dark:border-slate-800">
        <p class="text-sm text-slate-500 dark:text-slate-400">
          No previous enrollments for this student.
        </p>
      </div>
    `;

    return;
  }

  container.innerHTML = history
    .map(
      (item) => `
        <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-800">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <p class="font-mono text-xs font-semibold text-slate-700 dark:text-slate-300">
                ${escapeHtml(item.reference_number || "—")}
              </p>

              <p class="mt-1 text-sm font-semibold text-slate-900 dark:text-white">
                ${escapeHtml(item.course || "—")}
              </p>

              <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                ${escapeHtml(item.package || "—")}
              </p>
            </div>

            <span class="${
              item.active
                ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400"
                : "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400"
            } inline-flex shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-semibold">
              ${item.active ? "Active" : "Inactive"}
            </span>
          </div>

          <div class="mt-3 grid grid-cols-2 gap-3 text-xs">
            <div>
              <p class="text-slate-400">Classes</p>
              <p class="mt-1 font-medium text-slate-700 dark:text-slate-300">
                ${item.total_classes}
              </p>
            </div>

            <div>
              <p class="text-slate-400">Remaining</p>
              <p class="mt-1 font-medium text-slate-700 dark:text-slate-300">
                ${item.classes_remaining}
              </p>
            </div>
          </div>
        </div>
      `,
    )
    .join("");
}

function renderEnrollmentClassHistory(classes) {
  const container = document.getElementById("detailsClassHistory");

  if (!container) {
    return;
  }

  container.innerHTML = "";

  if (!Array.isArray(classes) || classes.length === 0) {
    container.innerHTML = `
      <div class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-500">
        No classes have been scheduled for this enrollment.
      </div>
    `;
    return;
  }

  classes.forEach((classItem) => {
    const row = document.createElement("div");
    row.className = "rounded-lg border border-gray-200 bg-white px-4 py-3";

    const classDate = classItem.class_date ? new Date(classItem.class_date).toLocaleDateString() : "—";

    const startTime = classItem.start_time
      ? new Date(classItem.start_time).toLocaleTimeString([], {
          hour: "numeric",
          minute: "2-digit",
        })
      : "—";

    const endTime = classItem.end_time
      ? new Date(classItem.end_time).toLocaleTimeString([], {
          hour: "numeric",
          minute: "2-digit",
        })
      : "—";

    let statusLabel = "Scheduled";
    let statusClass = "bg-gray-100 text-gray-700";

    switch (classItem.status) {
      case "present":
        statusLabel = "Present";
        statusClass = "bg-green-100 text-green-700";
        break;
      case "absent":
        statusLabel = "Absent";
        statusClass = "bg-red-100 text-red-700";
        break;
      case "cancelled":
        statusLabel = "Cancelled";
        statusClass = "bg-gray-100 text-gray-500";
        break;
    }

    const creditLabel = classItem.credit_refunded ? "Credit refunded" : "Credit consumed";

    row.innerHTML = `
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="text-sm font-medium text-gray-900">
            ${escapeHtml(classDate)}
          </div>
          <div class="text-xs text-gray-500">
            ${escapeHtml(startTime)} – ${escapeHtml(endTime)}
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium ${statusClass}">
            ${statusLabel}
          </span>
          <span class="text-xs text-gray-500">
            ${escapeHtml(creditLabel)}
          </span>
        </div>
      </div>
    `;

    container.appendChild(row);
  });

  refreshIcons(container);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function openRenewEnrollmentModal() {
  const modal = document.getElementById("renewEnrollmentModal");

  if (!modal) {
    return;
  }

  modal.classList.remove("hidden");
  modal.setAttribute("aria-hidden", "false");
  document.body.classList.add("overflow-hidden");

  refreshIcons();
}

function closeRenewEnrollmentModal() {
  const modal = document.getElementById("renewEnrollmentModal");

  if (!modal) {
    return;
  }

  modal.classList.add("hidden");
  modal.setAttribute("aria-hidden", "true");
  document.body.classList.remove("overflow-hidden");

  resetRenewEnrollmentForm();
}

function resetRenewEnrollmentForm() {
  const form = document.getElementById("renewEnrollmentForm");
  const packageSummary = document.getElementById("renewPackageSummary");
  const errorBox = document.getElementById("renewEnrollmentError");
  const fileName = document.getElementById("renewContractFileName");

  if (form) {
    form.reset();
  }

  setEnrollmentDetailsText("renewEnrollmentID", "");

  const studentName = document.getElementById("renewStudentName");

  if (studentName) {
    studentName.textContent = currentRenewalStudentName || "—";
  }

  packageSummary?.classList.add("hidden");

  if (errorBox) {
    errorBox.textContent = "";
    errorBox.classList.add("hidden");
  }

  if (fileName) {
    fileName.textContent = "";
    fileName.classList.add("hidden");
  }

  const submitButton = document.getElementById("submitRenewEnrollment");

  if (submitButton) {
    submitButton.disabled = false;
    submitButton.innerHTML = `
      <i data-lucide="refresh-cw" class="h-4 w-4"></i>
      Renew Enrollment
    `;
  }

  refreshIcons();
}

function updateRenewalPackageSummary() {
  const packageSelect = document.getElementById("renewPackageID");
  const summary = document.getElementById("renewPackageSummary");
  const summaryName = document.getElementById("renewPackageSummaryName");
  const summaryText = document.getElementById("renewPackageSummaryText");

  if (!packageSelect || !summary || !summaryName || !summaryText) {
    return;
  }

  const option = packageSelect.options[packageSelect.selectedIndex];

  if (!option || !option.value) {
    summary.classList.add("hidden");
    return;
  }

  const numberOfClasses = Number(option.dataset.numberOfClasses || 0);
  const numberOfFreeClasses = Number(option.dataset.numberOfFreeClasses || 0);
  const total = numberOfClasses + numberOfFreeClasses;
  const price = option.dataset.price || "";
  const validFrom = option.dataset.validFrom || "";
  const validUntil = option.dataset.validUntil || "";

  if (!Number.isFinite(total) || total <= 0) {
    summary.classList.add("hidden");
    return;
  }

  const classLabel = total === 1 ? "class" : "classes";
  const details = [`${total} ${classLabel} included`];

  if (numberOfFreeClasses > 0) {
    details.push(`${numberOfFreeClasses} free`);
  }

  if (price) {
    details.push(`Price: ${price}`);
  }

  if (validFrom && validUntil) {
    details.push(`Valid ${formatPackageDate(validFrom)} – ${formatPackageDate(validUntil)}`);
  }

  summaryName.textContent = option.textContent.trim();
  summaryText.textContent = details.join(" · ");
  summary.classList.remove("hidden");

  refreshIcons();
}

function setupRenewEnrollmentState(enrollment) {
  const button = document.getElementById("renewEnrollment");
  const unavailable = document.getElementById("renewEnrollmentUnavailable");

  if (!button || !unavailable) {
    return;
  }

  const canRenew = Boolean(enrollment.active);

  button.disabled = !canRenew;
  unavailable.classList.toggle("hidden", canRenew);

  if (!canRenew) {
    unavailable.textContent = "This enrollment is inactive and cannot be renewed.";
  }
}

function setupRenewEnrollmentHandler() {
  const button = document.getElementById("renewEnrollment");

  if (!button) {
    return;
  }

  button.addEventListener("click", openRenewEnrollment);
}

function openRenewEnrollment() {
  if (!currentEnrollmentDetailsID || !currentEnrollmentDetailsActive) {
    return;
  }

  const enrollmentID = currentEnrollmentDetailsID;

  const formID = document.getElementById("renewEnrollmentID");
  const studentName = document.getElementById("renewStudentName");
  const courseSelect = document.getElementById("renewCourseID");
  const packageSelect = document.getElementById("renewPackageID");
  const contractInput = document.getElementById("renewContract");

  if (formID) {
    formID.value = String(enrollmentID);
  }

  if (studentName) {
    studentName.textContent = currentRenewalStudentName || "—";
  }

  if (courseSelect) {
    courseSelect.value = "";
  }

  if (packageSelect) {
    packageSelect.value = "";
  }

  if (contractInput) {
    contractInput.value = "";
  }

  updateRenewalPackageSummary();
  openRenewEnrollmentModal();
}

async function submitRenewEnrollment(event) {
  event.preventDefault();

  const form = event.currentTarget;
  const errorBox = document.getElementById("renewEnrollmentError");
  const submitButton = document.getElementById("submitRenewEnrollment");

  if (!form || !submitButton) {
    return;
  }

  if (errorBox) {
    errorBox.textContent = "";
    errorBox.classList.add("hidden");
  }

  const formData = new FormData(form);

  const enrollmentID = String(formData.get("enrollment_id") || "").trim();
  const courseID = String(formData.get("course_id") || "").trim();
  const packageID = String(formData.get("package_id") || "").trim();
  const contract = formData.get("Contract");

  if (!enrollmentID) {
    showRenewEnrollmentError("Invalid enrollment ID.");
    return;
  }

  if (!courseID) {
    showRenewEnrollmentError("Select a course.");
    return;
  }

  if (!packageID) {
    showRenewEnrollmentError("Select a package.");
    return;
  }

  if (!(contract instanceof File) || contract.size === 0) {
    showRenewEnrollmentError("Upload a new contract.");
    return;
  }

  const packageSelect = document.getElementById("renewPackageID");
  const selectedPackage = packageSelect?.options[packageSelect.selectedIndex];

  const numberOfClasses = Number(selectedPackage?.dataset.numberOfClasses || 0);

  const numberOfFreeClasses = Number(selectedPackage?.dataset.numberOfFreeClasses || 0);

  const total = numberOfClasses + numberOfFreeClasses;

  if (!Number.isFinite(total) || total <= 0) {
    showRenewEnrollmentError("The selected package has no available classes.");
    return;
  }

  submitButton.disabled = true;
  submitButton.innerHTML = `
    <i data-lucide="loader-circle" class="h-4 w-4 animate-spin"></i>
    Renewing...
  `;
  refreshIcons();

  try {
    const response = await fetch("/enrollment/renew/", {
      method: "POST",
      body: formData,
      headers: {
        Accept: "application/json",
      },
    });

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (!response.ok || data.status !== "ok") {
      throw new Error(data.message || "Could not renew the enrollment.");
    }

    closeRenewEnrollmentModal();

    await reloadEnrollmentTable();

    await Swal.fire({
      icon: "success",
      title: "Enrollment renewed",
      html: `
        <div class="text-center">
          <p class="text-sm">
            A new enrollment and invoice were created successfully.
          </p>
          <p class="mt-3 font-mono text-sm font-semibold">
            ${escapeHtml(data.reference_number || "—")}
          </p>
        </div>
      `,
      confirmButtonText: "View New Enrollment",
    });

    if (data.enrollment_id) {
      await showEnrollmentDetails(data.enrollment_id);
    }
  } catch (error) {
    console.error("Error renewing enrollment:", error);

    showRenewEnrollmentError(error.message || "Could not renew the enrollment.");
  } finally {
    submitButton.disabled = false;
    submitButton.innerHTML = `
      <i data-lucide="refresh-cw" class="h-4 w-4"></i>
      Renew Enrollment
    `;
    refreshIcons();
  }
}

function showRenewEnrollmentError(message) {
  const errorBox = document.getElementById("renewEnrollmentError");

  if (!errorBox) {
    return;
  }

  errorBox.textContent = message || "Could not renew the enrollment.";

  errorBox.classList.remove("hidden");
}

function setupRenewalForm() {
  const form = document.getElementById("renewEnrollmentForm");
  const packageSelect = document.getElementById("renewPackageID");
  const contractInput = document.getElementById("renewContract");
  const fileName = document.getElementById("renewContractFileName");
  const closeButton = document.getElementById("closeRenewEnrollment");
  const cancelButton = document.getElementById("cancelRenewEnrollment");
  const backdrop = document.getElementById("renewEnrollmentBackdrop");

  form?.addEventListener("submit", submitRenewEnrollment);

  packageSelect?.addEventListener("change", updateRenewalPackageSummary);

  contractInput?.addEventListener("change", () => {
    const file = contractInput.files?.[0];

    if (!file) {
      fileName?.classList.add("hidden");

      if (fileName) {
        fileName.textContent = "";
      }

      return;
    }

    if (fileName) {
      fileName.textContent = `${file.name} · ${formatFileSize(file.size)}`;
      fileName.classList.remove("hidden");
    }
  });

  closeButton?.addEventListener("click", closeRenewEnrollmentModal);

  cancelButton?.addEventListener("click", closeRenewEnrollmentModal);

  backdrop?.addEventListener("click", closeRenewEnrollmentModal);

  document.addEventListener("keydown", (event) => {
    const modal = document.getElementById("renewEnrollmentModal");

    if (event.key === "Escape" && modal && !modal.classList.contains("hidden")) {
      closeRenewEnrollmentModal();
    }
  });
}

function setupEnrollmentDetails() {
  const modal = document.getElementById("enrollmentDetailsModal");
  const loading = document.getElementById("enrollmentDetailsLoading");
  const content = document.getElementById("enrollmentDetailsContent");
  const closeButton = document.getElementById("closeEnrollmentDetails");
  const backdrop = document.getElementById("enrollmentDetailsBackdrop");
  const deactivateButton = document.getElementById("deactivateEnrollment");
  const deactivateUnavailable = document.getElementById("deactivateEnrollmentUnavailable");
  const viewInvoiceButton = document.getElementById(
    "viewEnrollmentInvoice",
  );

  viewInvoiceButton?.addEventListener(
    "click",
    viewEnrollmentInvoice,
  );

  if (deactivateButton) {
    deactivateButton.disabled = true;
  }

  if (deactivateUnavailable) {
    deactivateUnavailable.classList.add("hidden");
  }

  if (!modal || !loading || !content) {
    return;
  }

  document.addEventListener("click", (event) => {
    const button = event.target.closest(".view-enrollment-details");

    if (!button) {
      return;
    }

    const enrollmentID = button.dataset.enrollmentId;

    if (!enrollmentID) {
      return;
    }

    showEnrollmentDetails(enrollmentID);
  });

  closeButton?.addEventListener("click", closeEnrollmentDetailsModal);

  backdrop?.addEventListener("click", closeEnrollmentDetailsModal);

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !modal.classList.contains("hidden")) {
      closeEnrollmentDetailsModal();
    }
  });

  setupChangeCourseHandler();
  setupDeactivateEnrollmentHandler();
  setupRenewEnrollmentHandler();
  setupRenewalForm();
}

function setupDeactivateEnrollmentState(enrollment) {
  const button = document.getElementById("deactivateEnrollment");

  const unavailable = document.getElementById("deactivateEnrollmentUnavailable");

  if (!button || !unavailable) {
    return;
  }

  const isActive = Boolean(enrollment.active);

  button.disabled = !isActive;

  unavailable.classList.toggle("hidden", isActive);

  if (!isActive) {
    unavailable.textContent = "This enrollment is already inactive.";
  }
}

function setupDeactivateEnrollmentHandler() {
  const button = document.getElementById("deactivateEnrollment");

  if (!button) {
    return;
  }

  button.addEventListener("click", deactivateEnrollment);
}

async function deactivateEnrollment() {
  const button = document.getElementById("deactivateEnrollment");

  if (!button || !currentEnrollmentDetailsID) {
    return;
  }

  if (!currentEnrollmentDetailsActive) {
    return;
  }

  const confirmation = await Swal.fire({
    icon: "warning",
    title: "Deactivate enrollment?",
    text: "The enrollment will become inactive and its remaining classes will no longer be usable through this enrollment.",
    showCancelButton: true,
    confirmButtonText: "Deactivate",
    cancelButtonText: "Cancel",
    reverseButtons: true,
    focusCancel: true,
  });

  if (!confirmation.isConfirmed) {
    return;
  }

  button.disabled = true;

  try {
    const formData = new URLSearchParams();

    formData.set("enrollment_id", String(currentEnrollmentDetailsID));

    const response = await fetch("/enrollment/deactivate/", {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Accept: "application/json",
      },
      body: formData.toString(),
    });

    const contentType = response.headers.get("content-type") || "";

    if (!contentType.includes("application/json")) {
      throw new Error("The server returned an unexpected response.");
    }

    const data = await response.json();

    if (!response.ok || data.status !== "ok") {
      throw new Error(data.message || "Could not deactivate the enrollment.");
    }

    await reloadEnrollmentTable();
    await showEnrollmentDetails(currentEnrollmentDetailsID);

    await Swal.fire({
      icon: "success",
      title: "Enrollment deactivated",
      text: "The enrollment is now inactive.",
      timer: 1600,
      showConfirmButton: false,
    });
  } catch (error) {
    console.error("Error deactivating enrollment:", error);

    button.disabled = false;

    await Swal.fire({
      icon: "error",
      title: "Could not deactivate enrollment",
      text: error.message || "Could not deactivate the enrollment. Please try again.",
    });
  }
}
