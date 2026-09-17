document.addEventListener("DOMContentLoaded", function () {
  // --- ADD COURSE MODAL ---
  const modal = document.getElementById("addCourseModal");
  const openButton = document.getElementById("openAddCourseModal");
  const closeButton = document.getElementById("closeAddCourseModal");
  const cancelButton = document.getElementById("cancelAddCourseModal");
  const backdrop = document.getElementById("addCourseBackdrop");

  function openModal() {
    modal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
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

  const addCourseForm = document.getElementById("addCourseForm");
  addCourseForm.addEventListener("submit", function (e) {
    e.preventDefault();
    const formData = new FormData(addCourseForm);

    fetch("/course/", {
      method: "POST",
      body: formData,
    })
      .then((response) => response.json())
      .then((data) => {
        if (data.status === "ok") {
          addCourseForm.reset();
          closeModal();
          reloadCourseTable();
        }
      })
      .catch((error) => console.error("Error: ", error));
  });

  function reloadCourseTable() {
    fetch("/course/")
      .then((response) => response.text())
      .then((html) => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, "text/html");
        const newTable = doc.getElementById("coursesTable");
        const currentTable = document.getElementById("coursesTable");

        if (newTable && currentTable) {
          currentTable.innerHTML = newTable.innerHTML;
          if (typeof lucide !== "undefined") {
            lucide.createIcons();
          }
        }
      })
      .catch((err) => console.error("Failed to refresh table: ", err));
  }

  // --- EDIT COURSE MODAL ---
  const editModal = document.getElementById("editCourseModal");
  const editForm = document.getElementById("editCourseForm");

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
    .getElementById("closeEditCourseModal")
    .addEventListener("click", closeEditModal);
  document
    .getElementById("cancelEditCourseModal")
    .addEventListener("click", closeEditModal);
  document
    .getElementById("editCourseBackdrop")
    .addEventListener("click", closeEditModal);

  // Dynamic delegation for Edit & Materials buttons in table
  document
    .getElementById("coursesTable")
    .addEventListener("click", function (e) {
      const editBtn = e.target.closest(".edit-course-btn");
      if (editBtn) {
        document.getElementById("editCourseId").value = editBtn.dataset.id;
        document.getElementById("editTitle").value = editBtn.dataset.title;
        document.getElementById("editDescription").value =
          editBtn.dataset.description;
        document.getElementById("editLevelID").value = editBtn.dataset.level;
        document.getElementById("editActive").value =
          editBtn.dataset.active === "true" ? "true" : "false";
        openEditModal();
        return;
      }

      const materialsBtn = e.target.closest(".manage-materials-btn");
      if (materialsBtn) {
        const courseId = materialsBtn.dataset.courseId;
        const courseTitle = materialsBtn.dataset.courseTitle;

        document.getElementById("materialCourseID").value = courseId;
        document.getElementById("materialsModalSubtitle").textContent =
          `Managing materials for: ${courseTitle}`;

        loadCourseMaterials(courseId);
        materialsModal.classList.remove("hidden");
        document.body.classList.add("overflow-hidden");
      }
    });

  editForm.addEventListener("submit", function (e) {
    e.preventDefault();
    const formData = new FormData(editForm);
    const courseId = formData.get("id");

    fetch(`/course/?id=${courseId}`, {
      method: "PUT",
      body: formData,
    })
      .then((response) => response.json())
      .then((data) => {
        if (data.status === "ok") {
          closeEditModal();
          reloadCourseTable();
        } else {
          alert("Error: " + data.message);
        }
      })
      .catch((error) => console.error("Error updating course: ", error));
  });

  // --- MATERIALS MODAL ---
  const materialsModal = document.getElementById("materialsModal");
  const closeMaterialsBtn = document.getElementById("closeMaterialsModal");
  const materialsBackdrop = document.getElementById("materialsBackdrop");
  const uploadMaterialForm = document.getElementById("uploadMaterialForm");
  const assignedMaterialsList = document.getElementById(
    "assignedMaterialsList",
  );

  function closeMaterialsModal() {
    materialsModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
    uploadMaterialForm.reset();
  }

  closeMaterialsBtn.addEventListener("click", closeMaterialsModal);
  materialsBackdrop.addEventListener("click", closeMaterialsModal);

  function loadCourseMaterials(courseId) {
    fetch(`/course-materials/?course_id=${courseId}`)
      .then((res) => res.json())
      .then((data) => {
        assignedMaterialsList.innerHTML = "";

        if (!data || data.length === 0) {
          assignedMaterialsList.innerHTML = `<li class="text-sm text-gray-500">No materials attached yet.</li>`;
          return;
        }

        data.forEach((item) => {
          assignedMaterialsList.innerHTML += `
          <li class="flex items-center justify-between rounded-xl border border-gray-100 p-3 dark:border-gray-800">
            <div class="flex items-center gap-2">
              <i data-lucide="file-text" class="h-4 w-4 text-red-500"></i>
              <span class="text-sm font-medium text-gray-800 dark:text-gray-200">${item.Material.Name}</span>
            </div>
            <div class="flex items-center gap-3">
              <a href="${item.Material.File}" target="_blank" rel="noopener noreferrer"
                class="text-xs font-medium text-blue-600 hover:underline dark:text-blue-400">
                View PDF
              </a>
              <button type="button" onclick="deleteCourseMaterial(${item.ID}, ${courseId})" class="text-xs text-red-500 hover:underline">
                Remove
              </button>
            </div>
          </li>`;
        });

        if (typeof lucide !== "undefined") {
          lucide.createIcons();
        }
      })
      .catch((err) => console.error("Error loading materials:", err));
  }

  // Attach load function to window so delete handler can refresh list
  window.loadCourseMaterials = loadCourseMaterials;

  uploadMaterialForm.addEventListener("submit", function (e) {
    e.preventDefault();
    const formData = new FormData(uploadMaterialForm);
    const courseId = formData.get("courseID");

    fetch("/course-materials/", {
      method: "POST",
      body: formData,
    })
      .then((res) => res.json())
      .then((data) => {
        if (data.status === "ok") {
          uploadMaterialForm.reset();
          document.getElementById("materialCourseID").value = courseId;
          loadCourseMaterials(courseId);
        } else {
          alert("Upload failed: " + data.message);
        }
      })
      .catch((err) => console.error("Error uploading file:", err));
  });
});

// Global Delete Handler for inline onclick calls
window.deleteCourseMaterial = function (id, courseId) {
  if (!confirm("Are you sure you want to remove this material?")) return;

  fetch(`/course-materials/?id=${id}`, {
    method: "DELETE",
  })
    .then((res) => res.json())
    .then((data) => {
      if (data.status === "ok") {
        window.loadCourseMaterials(courseId);
      } else {
        alert("Delete failed: " + data.message);
      }
    })
    .catch((err) => console.error("Error deleting material:", err));
};
