document.addEventListener("DOMContentLoaded", function () {
  const modal = document.getElementById("addStudentModal");
  const openButton = document.getElementById("openAddStudentModal");
  const closeButton = document.getElementById("closeAddStudentModal");
  const cancelButton = document.getElementById("cancelAddStudentModal");
  const backdrop = document.getElementById("addStudentBackdrop");

  const credsModal = document.getElementById("studentCredsModal");
  const credsBackdrop = document.getElementById("studentCredsBackdrop");
  const closeCredsButton = document.getElementById("closeStudentCredsModal");
  const confirmCredsButton = document.getElementById(
    "confirmStudentCredsModal",
  );
  const copyAllCredsButton = document.getElementById("copyAllCredsButton");
  const credsList = document.getElementById("studentCredsList");

  function openCredsModal(creds) {
    credsList.innerHTML = "";
    let plainText = "";

    Object.entries(creds || {}).forEach(([key, value]) => {
      const label = key
        .replace(/([A-Z])/g, " $1")
        .replace(/^./, (s) => s.toUpperCase())
        .trim();
      plainText += `${label}: ${value}\n`;

      const row = document.createElement("div");
      row.className =
        "flex items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 py-2 dark:border-gray-700";
      row.innerHTML = `
      <div>
        <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">${label}</p>
        <p class="font-mono text-sm text-gray-900 dark:text-white">${value}</p>
      </div>
      <button type="button" class="copy-cred-btn rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-gray-800 dark:hover:text-gray-200" data-value="${value}" aria-label="Copy ${label}">
        <i data-lucide="copy" class="h-4 w-4"></i>
      </button>
    `;
      credsList.appendChild(row);
    });

    copyAllCredsButton.dataset.value = plainText.trim();
    credsModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");

    if (typeof lucide !== "undefined") {
      lucide.createIcons();
    }
  }

  function closeCredsModal() {
    credsModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  closeCredsButton.addEventListener("click", closeCredsModal);
  confirmCredsButton.addEventListener("click", closeCredsModal);
  credsBackdrop.addEventListener("click", closeCredsModal);

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && !credsModal.classList.contains("hidden")) {
      closeCredsModal();
    }
  });

  credsList.addEventListener("click", function (e) {
    const btn = e.target.closest(".copy-cred-btn");
    if (!btn) return;
    navigator.clipboard.writeText(btn.dataset.value).then(() => {
      const icon = btn.querySelector("i");
      if (icon) {
        icon.setAttribute("data-lucide", "check");
        lucide.createIcons();
        setTimeout(() => {
          icon.setAttribute("data-lucide", "copy");
          lucide.createIcons();
        }, 1200);
      }
    });
  });

  copyAllCredsButton.addEventListener("click", function () {
    navigator.clipboard.writeText(copyAllCredsButton.dataset.value || "");
  });

  function openModal() {
    modal.classList.remove("hidden");
  }

  function closeModal() {
    modal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  openButton.addEventListener("click", openModal);
  closeButton.addEventListener("click", closeModal);
  cancelButton.addEventListener("click", closeModal);
  backdrop.addEventListener("click", closeModal);

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && !modal.classList.contains("hidden")) {
      closeModal();
    }
  });

  const addStudentForm = document.getElementById("addStudentForm");
  addStudentForm.addEventListener("submit", function (e) {
    e.preventDefault();

    const formData = new FormData(addStudentForm);
    fetch("/student/", {
      method: "POST",
      body: formData,
    })
      .then((response) => response.json())
      .then((data) => {
        console.log(data);
        if (data.status == "ok") {
          addStudentForm.reset();
          closeModal();
          reloadStudentTable();
          openCredsModal(data.creds);
        } else {
          console.error('Save failed: ', data);
        }
      })
      .catch((error) => console.error("Error: ", error));
  });

  // Reload student table
  function reloadStudentTable() {
    fetch("/student/")
      .then((response) => response.text())
      .then((html) => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, "text/html");
        const newTable = doc.getElementById("studentsTable");
        const currentTable = document.getElementById("studentsTable");

        if (newTable && currentTable) {
          currentTable.innerHTML = newTable.innerHTML;
          if (typeof lucide !== "undefined") {
            lucide.createIcons();
          }
        }
      })
      .catch((err) => console.error("Failed to refresh table: ", err));
  }

  const editModal = document.getElementById("editStudentModal");
  const editForm = document.getElementById("editStudentForm");

  function openEditModal() {
    editModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeEditModal() {
    editModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
    editForm.reset();
  }

  document
    .getElementById("closeEditStudentModal")
    .addEventListener("click", closeEditModal);
  document
    .getElementById("cancelEditStudentModal")
    .addEventListener("click", closeEditModal);
  document
    .getElementById("editStudentBackdrop")
    .addEventListener("click", closeEditModal);

  // Dynamic delegation for Edit & Materials buttons in table
  document
    .getElementById("studentsTable")
    .addEventListener("click", function (e) {
      const editBtn = e.target.closest(".edit-student-btn");
      if (editBtn) {
        document.getElementById("editStudentId").value = editBtn.dataset.id;
        document.getElementById("editFirstName").value = editBtn.dataset.firstname;
        document.getElementById("editLastName").value =
          editBtn.dataset.lastname;
        document.getElementById("editWeChatId").value = editBtn.dataset.wechat;
        document.getElementById("editEmail").value = editBtn.dataset.email;
        openEditModal();
        return;
      }
    });

  editForm.addEventListener("submit", function (e) {
    e.preventDefault();
    const formData = new FormData(editForm);
    const studentId = formData.get("id");

    fetch(`/student/?id=${studentId}`, {
      method: "PUT",
      body: formData,
    })
      .then((response) => response.json())
      .then((data) => {
        if (data.status === "ok") {
          closeEditModal();
          reloadStudentTable();
        } else {
          alert("Error: " + data.message);
        }
      })
      .catch((error) => console.error("Error updating course: ", error));
  });
  
});
