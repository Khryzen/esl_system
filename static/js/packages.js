// packages.js: Add / Edit handling for the Packages page.
// Loaded from packages.html with: <script src="/static/js/packages.js" defer></script>

(() => {
  "use strict";

  // Change this if your Go route is different.
  // Create -> POST   /package/
  // Edit   -> PUT    /package/?id=<id>
  const PACKAGE_ENDPOINT = "/package/";

  const modal = document.getElementById("packageModal");
  const form = document.getElementById("packageForm");
  if (!modal || !form) return;

  const title = document.getElementById("packageModalTitle");
  const submitBtn = document.getElementById("packageSubmitBtn");
  const idInput = document.getElementById("packageId");
  const activeInput = document.getElementById("packageActive");
  const validFrom = document.getElementById("packageValidFrom");
  const validUntil = document.getElementById("packageValidUntil");
  const imageInput = document.getElementById("packageImage");
  const imagePreview = document.getElementById("packageImagePreview");

  // form input id -> data-* key on the Edit button (data-free-classes => freeClasses)
  const fieldMap = {
    packageName: "name",
    packageClasses: "classes",
    packageFreeClasses: "freeClasses",
    packageDuration: "duration",
    packagePrice: "price",
    packageValidFrom: "validFrom",
    packageValidUntil: "validUntil",
  };

  let currentImage = ""; // image already saved on the server (edit mode)
  let previewUrl = null; // object URL of a newly chosen file

  /* ---------- Image preview ---------- */

  function showPreview(src, isObjectUrl = false) {
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    previewUrl = isObjectUrl ? src : null;

    if (src) {
      imagePreview.src = src;
      imagePreview.classList.remove("hidden");
    } else {
      imagePreview.removeAttribute("src");
      imagePreview.classList.add("hidden");
    }
  }

  imageInput.addEventListener("change", () => {
    const file = imageInput.files[0];
    if (file) {
      showPreview(URL.createObjectURL(file), true);
    } else {
      showPreview(currentImage); // selection cleared: fall back to the saved image
    }
  });

  /* ---------- Date validation ---------- */

  function validateDates() {
    const invalid =
      validFrom.value && validUntil.value && validUntil.value < validFrom.value;
    validUntil.setCustomValidity(
      invalid ? "Valid Until can't be earlier than Valid From." : "",
    );
  }

  validFrom.addEventListener("input", validateDates);
  validUntil.addEventListener("input", validateDates);

  /* ---------- Open / close ---------- */

  function openModal() {
    validateDates();
    modal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
    document.getElementById("packageName").focus();
  }

  function closeModal() {
    modal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  function openAddModal() {
    form.reset(); // restores defaults: free classes = 0, active = checked
    idInput.value = "";
    currentImage = "";
    showPreview("");
    title.textContent = "Add Package";
    submitBtn.textContent = "Create Package";
    openModal();
  }

  function openEditModal(btn) {
    const d = btn.dataset;

    form.reset();
    idInput.value = d.id;
    for (const [inputId, key] of Object.entries(fieldMap)) {
      document.getElementById(inputId).value = d[key] ?? "";
    }
    activeInput.checked = d.active === "true";

    currentImage = d.image || "";
    showPreview(currentImage);

    title.textContent = "Edit Package";
    submitBtn.textContent = "Update Package";
    openModal();
  }

  /* ---------- Refresh the table without a full reload ---------- */

  async function reloadPackagesTable() {
    try {
      const response = await fetch(window.location.href);
      const html = await response.text();
      const doc = new DOMParser().parseFromString(html, "text/html");

      const freshBody = doc.querySelector("#packagesTable tbody");
      const currentBody = document.querySelector("#packagesTable tbody");
      if (!freshBody || !currentBody)
        throw new Error("Packages table not found in response");

      currentBody.replaceWith(freshBody);
      if (window.lucide) window.lucide.createIcons(); // re-draw icons in the new rows
    } catch (error) {
      console.error("Error refreshing packages table:", error);
      window.location.reload();
    }
  }

  /* ---------- Submit (create or update) ---------- */

  form.addEventListener("submit", async (e) => {
    e.preventDefault();

    const isEdit = idInput.value !== "";
    const formData = new FormData(form);

    // Unchecked checkboxes aren't sent at all, and a checked one sends "on",
    // which Go's strconv.ParseBool rejects. Always send an explicit true/false.
    formData.set("active", activeInput.checked ? "true" : "false");

    // Skip the empty file part so an edit without a new image keeps the old one.
    const file = formData.get("image");
    if (file instanceof File && file.size === 0) formData.delete("image");

    const url = isEdit
      ? `${PACKAGE_ENDPOINT}?id=${encodeURIComponent(idInput.value)}`
      : PACKAGE_ENDPOINT;

    submitBtn.disabled = true;
    try {
      const response = await fetch(url, {
        method: isEdit ? "PUT" : "POST",
        body: formData,
      });
      const data = await response.json();

      if (data.status === "ok") {
        closeModal();
        await reloadPackagesTable();
      } else {
        alert("Error: " + (data.message || "Something went wrong."));
      }
    } catch (error) {
      console.error(
        `Error ${isEdit ? "updating" : "creating"} package:`,
        error,
      );
      alert("Could not save the package. Please try again.");
    } finally {
      submitBtn.disabled = false;
    }
  });

  /* ---------- Event wiring ---------- */

  document
    .getElementById("openAddPackageModal")
    ?.addEventListener("click", openAddModal);

  // Delegated, so Edit buttons keep working after the table body is swapped.
  document.addEventListener("click", (e) => {
    const editBtn = e.target.closest(".edit-package-btn");
    if (editBtn) openEditModal(editBtn);
  });

  // Backdrop, X button and Cancel button all carry data-close-modal.
  modal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-modal]")) closeModal();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !modal.classList.contains("hidden")) closeModal();
  });
})();
