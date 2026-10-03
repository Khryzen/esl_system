document.addEventListener("DOMContentLoaded", function () {
  const addModal = document.getElementById("addStudentModal");
  const addForm = document.getElementById("addStudentForm");
  const addButton = document.getElementById("openAddStudentModal");
  const closeAddButton = document.getElementById("closeAddStudentModal");
  const cancelAddButton = document.getElementById("cancelAddStudentModal");
  const addBackdrop = document.getElementById("addStudentBackdrop");
  const addSaveButton = document.getElementById("studentSaveBtn");
  const addError = document.getElementById("addStudentError");

  const editModal = document.getElementById("editStudentModal");
  const editForm = document.getElementById("editStudentForm");
  const closeEditButton = document.getElementById("closeEditStudentModal");
  const cancelEditButton = document.getElementById("cancelEditStudentModal");
  const editBackdrop = document.getElementById("editStudentBackdrop");
  const updateButton = document.getElementById("updateStudentBtn");
  const editError = document.getElementById("editStudentError");

  const studentsTable = document.getElementById("studentsTable");
  const studentsTableBody = document.getElementById("studentsTableBody");

  const searchInput = document.getElementById("studentSearch");
  const clearSearchButton = document.getElementById("clearStudentSearch");
  const searchEmpty = document.getElementById("studentSearchEmpty");
  const visibleCount = document.getElementById("studentVisibleCount");

  const credsModal = document.getElementById("studentCredsModal");
  const credsBackdrop = document.getElementById("studentCredsBackdrop");
  const closeCredsButton = document.getElementById("closeStudentCredsModal");
  const confirmCredsButton = document.getElementById(
    "confirmStudentCredsModal",
  );
  const copyAllCredsButton = document.getElementById("copyAllCredsButton");
  const credsList = document.getElementById("studentCredsList");

  function refreshIcons() {
    if (typeof lucide !== "undefined") {
      lucide.createIcons();
    }
  }

  function lockBody() {
    document.body.classList.add("overflow-hidden");
  }

  function unlockBody() {
    if (
      addModal.classList.contains("hidden") &&
      editModal.classList.contains("hidden") &&
      credsModal.classList.contains("hidden")
    ) {
      document.body.classList.remove("overflow-hidden");
    }
  }

  function showError(element, message) {
    element.textContent = message || "Something went wrong.";
    element.classList.remove("hidden");
  }

  function hideError(element) {
    element.textContent = "";
    element.classList.add("hidden");
  }

  function setButtonLoading(button, loading, loadingText, defaultText) {
    button.disabled = loading;

    if (loading) {
      button.dataset.originalText = defaultText;
      button.innerHTML = `
        <i data-lucide="loader-circle" class="h-4 w-4 animate-spin"></i>
        ${loadingText}
      `;
    } else {
      button.innerHTML = `
        <i data-lucide="${button === addSaveButton ? "user-plus" : "save"}" class="h-4 w-4"></i>
        ${defaultText}
      `;
    }

    refreshIcons();
  }

  // ------------------------------------------------------------
  // ADD STUDENT
  // ------------------------------------------------------------

  function openAddModal() {
    hideError(addError);
    addModal.classList.remove("hidden");
    lockBody();

    setTimeout(() => {
      document.getElementById("firstName")?.focus();
    }, 50);
  }

  function closeAddModal() {
    addModal.classList.add("hidden");
    addForm.reset();
    hideError(addError);
    unlockBody();
  }

  addButton.addEventListener("click", openAddModal);
  closeAddButton.addEventListener("click", closeAddModal);
  cancelAddButton.addEventListener("click", closeAddModal);
  addBackdrop.addEventListener("click", closeAddModal);

  addForm.addEventListener("submit", async function (event) {
    event.preventDefault();

    hideError(addError);

    const formData = new FormData(addForm);

    setButtonLoading(addSaveButton, true, "Creating...", "Create Student");

    try {
      const response = await fetch("/student/", {
        method: "POST",
        body: formData,
      });

      const contentType = response.headers.get("content-type") || "";

      if (!contentType.includes("application/json")) {
        throw new Error("The server returned an unexpected response.");
      }

      const data = await response.json();

      if (data.status !== "ok") {
        throw new Error(data.message || "Could not create the student.");
      }

      closeAddModal();

      await reloadStudentTable();

      if (data.creds) {
        openCredsModal(data.creds);
      }
    } catch (error) {
      console.error("Error creating student:", error);
      showError(addError, error.message);
    } finally {
      setButtonLoading(addSaveButton, false, "", "Create Student");
    }
  });

  // ------------------------------------------------------------
  // STUDENT CREDENTIALS
  // ------------------------------------------------------------

  let allCredentialsText = "";

  function openCredsModal(creds) {
    credsList.innerHTML = "";
    allCredentialsText = "";

    Object.entries(creds || {}).forEach(([key, value]) => {
      const label = key
        .replace(/([A-Z])/g, " $1")
        .replace(/^./, (character) => character.toUpperCase())
        .trim();

      allCredentialsText += `${label}: ${value}\n`;

      const row = document.createElement("div");

      row.className =
        "flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 px-4 py-3 dark:border-slate-700 dark:bg-slate-800/60";

      row.innerHTML = `
        <div class="min-w-0">
          <p class="text-[11px] font-semibold uppercase tracking-wide text-slate-400 dark:text-slate-500">
            ${escapeHtml(label)}
          </p>

          <p class="mt-1 truncate font-mono text-sm font-medium text-slate-900 dark:text-white">
            ${escapeHtml(String(value))}
          </p>
        </div>

        <button
          type="button"
          class="copy-cred-btn shrink-0 rounded-lg p-2 text-slate-400 transition hover:bg-white hover:text-slate-700 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          data-value="${escapeHtml(String(value))}"
          aria-label="Copy ${escapeHtml(label)}"
        >
          <i data-lucide="copy" class="h-4 w-4"></i>
        </button>
      `;

      credsList.appendChild(row);
    });

    credsModal.classList.remove("hidden");
    lockBody();
    refreshIcons();
  }

  function closeCredsModal() {
    credsModal.classList.add("hidden");
    unlockBody();
  }

  closeCredsButton.addEventListener("click", closeCredsModal);
  confirmCredsButton.addEventListener("click", closeCredsModal);
  credsBackdrop.addEventListener("click", closeCredsModal);

  credsList.addEventListener("click", async function (event) {
    const button = event.target.closest(".copy-cred-btn");

    if (!button) {
      return;
    }

    try {
      await navigator.clipboard.writeText(button.dataset.value || "");

      const icon = button.querySelector("i");

      if (icon) {
        icon.setAttribute("data-lucide", "check");
        refreshIcons();

        setTimeout(() => {
          icon.setAttribute("data-lucide", "copy");
          refreshIcons();
        }, 1200);
      }
    } catch (error) {
      console.error("Could not copy credential:", error);
    }
  });

  copyAllCredsButton.addEventListener("click", async function () {
    try {
      await navigator.clipboard.writeText(allCredentialsText.trim());

      const original = copyAllCredsButton.innerHTML;

      copyAllCredsButton.innerHTML = `
        <i data-lucide="check" class="h-4 w-4"></i>
        Copied
      `;

      refreshIcons();

      setTimeout(() => {
        copyAllCredsButton.innerHTML = original;
        refreshIcons();
      }, 1200);
    } catch (error) {
      console.error("Could not copy credentials:", error);
    }
  });

  // ------------------------------------------------------------
  // EDIT STUDENT
  // ------------------------------------------------------------

  function openEditModal(student) {
    hideError(editError);

    document.getElementById("editStudentId").value = student.id;
    document.getElementById("editFirstName").value = student.firstname;
    document.getElementById("editLastName").value = student.lastname;
    document.getElementById("editWeChatId").value = student.wechat;
    document.getElementById("editEmail").value = student.email;

    editModal.classList.remove("hidden");
    lockBody();

    setTimeout(() => {
      document.getElementById("editFirstName")?.focus();
    }, 50);
  }

  function closeEditModal() {
    editModal.classList.add("hidden");
    editForm.reset();
    hideError(editError);
    unlockBody();
  }

  closeEditButton.addEventListener("click", closeEditModal);
  cancelEditButton.addEventListener("click", closeEditModal);
  editBackdrop.addEventListener("click", closeEditModal);

  studentsTableBody.addEventListener("click", function (event) {
    const editButton = event.target.closest(".edit-student-btn");

    if (!editButton) {
      return;
    }

    openEditModal({
      id: editButton.dataset.id,
      firstname: editButton.dataset.firstname || "",
      lastname: editButton.dataset.lastname || "",
      wechat: editButton.dataset.wechat || "",
      email: editButton.dataset.email || "",
    });
  });

  editForm.addEventListener("submit", async function (event) {
    event.preventDefault();

    hideError(editError);

    const formData = new FormData(editForm);
    const studentID = formData.get("id");

    if (!studentID) {
      showError(editError, "Invalid student ID.");
      return;
    }

    setButtonLoading(updateButton, true, "Saving...", "Save Changes");

    try {
      const response = await fetch(`/student/?id=${studentID}`, {
        method: "PUT",
        body: formData,
      });

      const contentType = response.headers.get("content-type") || "";

      if (!contentType.includes("application/json")) {
        throw new Error("The server returned an unexpected response.");
      }

      const data = await response.json();

      if (data.status !== "ok") {
        throw new Error(data.message || "Could not update the student.");
      }

      closeEditModal();

      await reloadStudentTable();
    } catch (error) {
      console.error("Error updating student:", error);
      showError(editError, error.message);
    } finally {
      setButtonLoading(updateButton, false, "", "Save Changes");
    }
  });

  // ------------------------------------------------------------
  // TABLE REFRESH
  // ------------------------------------------------------------

  async function reloadStudentTable() {
    try {
      const response = await fetch("/student/", {
        method: "GET",
        headers: {
          Accept: "text/html",
        },
      });

      if (!response.ok) {
        throw new Error(`Could not reload students (${response.status}).`);
      }

      const html = await response.text();

      const parser = new DOMParser();
      const documentFragment = parser.parseFromString(html, "text/html");

      const newTable = documentFragment.getElementById("studentsTable");
      const currentTable = document.getElementById("studentsTable");

      if (!newTable || !currentTable) {
        throw new Error("Student table could not be refreshed.");
      }

      currentTable.innerHTML = newTable.innerHTML;

      refreshIcons();

      applyStudentSearch();
    } catch (error) {
      console.error("Failed to refresh student table:", error);
    }
  }

  // ------------------------------------------------------------
  // SEARCH
  // ------------------------------------------------------------

  function applyStudentSearch() {
    const query = searchInput.value.trim().toLowerCase();
    const rows = Array.from(document.querySelectorAll(".student-row"));

    let matches = 0;

    rows.forEach((row) => {
      const searchText = (
        row.dataset.search ||
        row.textContent ||
        ""
      ).toLowerCase();

      const visible = !query || searchText.includes(query);

      row.classList.toggle("hidden", !visible);

      if (visible) {
        matches++;
      }
    });

    visibleCount.textContent = matches;

    searchEmpty.classList.toggle("hidden", matches !== 0 || rows.length === 0);

    clearSearchButton.classList.toggle("hidden", !searchInput.value);
  }

  searchInput.addEventListener("input", applyStudentSearch);

  clearSearchButton.addEventListener("click", function () {
    searchInput.value = "";
    applyStudentSearch();
    searchInput.focus();
  });

  // ------------------------------------------------------------
  // ESCAPE KEY
  // ------------------------------------------------------------

  document.addEventListener("keydown", function (event) {
    if (event.key !== "Escape") {
      return;
    }

    if (!credsModal.classList.contains("hidden")) {
      closeCredsModal();
      return;
    }

    if (!editModal.classList.contains("hidden")) {
      closeEditModal();
      return;
    }

    if (!addModal.classList.contains("hidden")) {
      closeAddModal();
    }
  });

  // ------------------------------------------------------------
  // HTML ESCAPING
  // ------------------------------------------------------------

  function escapeHtml(value) {
    return String(value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // Initial state.
  applyStudentSearch();
  refreshIcons();
});
