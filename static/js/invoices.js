// invoices.js: filtering, Add Invoice, and Mark Paid for the Invoices page.

(() => {
  "use strict";

  const INVOICE_ENDPOINT = window.location.pathname;

  /* ---------- Filter tabs ---------- */

  const filterBtns = document.querySelectorAll("[data-invoice-filter]");
  const rows = () =>
    document.querySelectorAll("#invoicesTable tbody tr[data-paid]");

  function applyFilterStyles(activeBtn) {
    const active = "bg-gray-900 text-white dark:bg-white dark:text-gray-900";
    const inactive =
      "text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800";
    filterBtns.forEach((btn) => {
      btn.className = `invoice-filter-btn rounded-md px-3 py-1.5 text-sm font-medium transition ${btn === activeBtn ? active : inactive}`;
    });
  }

  filterBtns.forEach((btn) => {
    btn.addEventListener("click", () => {
      applyFilterStyles(btn);
      const filter = btn.dataset.invoiceFilter; // "all" | "unpaid" | "paid"
      rows().forEach((row) => {
        const paid = row.dataset.paid === "true";
        const show =
          filter === "all" ||
          (filter === "unpaid" && !paid) ||
          (filter === "paid" && paid);
        row.classList.toggle("hidden", !show);
      });
    });
  });
  applyFilterStyles(document.querySelector('[data-invoice-filter="all"]'));

  /* ---------- Add Invoice modal ---------- */

  const invoiceModal = document.getElementById("invoiceModal");
  const invoiceForm = document.getElementById("invoiceForm");
  const invoiceSubmitBtn = document.getElementById("invoiceSubmitBtn");
  const enrollmentSelect = document.getElementById("invoiceEnrollment");
  const enrollmentEmpty = document.getElementById("invoiceEnrollmentEmpty");
  const amountInput = document.getElementById("invoiceAmount");
  const dueDateInput = document.getElementById("invoiceDueDate");

  function openInvoiceModal() {
    invoiceForm.reset();

    // No active enrollments to bill: the placeholder option is the only one present.
    const hasEnrollments = enrollmentSelect.options.length > 1;
    enrollmentEmpty.classList.toggle("hidden", hasEnrollments);
    enrollmentSelect.disabled = !hasEnrollments;
    invoiceSubmitBtn.disabled = !hasEnrollments;

    // Default due date: two weeks out. Staff can still change it before submitting.
    const twoWeeksOut = new Date();
    twoWeeksOut.setDate(twoWeeksOut.getDate() + 14);
    dueDateInput.value = twoWeeksOut.toISOString().slice(0, 10);

    invoiceModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
    enrollmentSelect.focus();
  }

  function closeInvoiceModal() {
    invoiceModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  document
    .getElementById("openAddInvoiceModal")
    .addEventListener("click", openInvoiceModal);

  // Prefills Amount from the chosen enrollment's package price. Left editable on
  // purpose — a manual invoice won't always match the package price exactly.
  enrollmentSelect.addEventListener("change", () => {
    const price = enrollmentSelect.selectedOptions[0]?.dataset.price;
    if (price) amountInput.value = price;
  });

  invoiceForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    invoiceSubmitBtn.disabled = true;
    try {
      const response = await fetch(INVOICE_ENDPOINT, {
        method: "POST",
        body: new FormData(invoiceForm),
      });
      const data = await response.json();

      if (data.status === "ok") {
        // A full reload, not a partial swap: creating an invoice also changes the
        // Outstanding Balance / Unpaid Invoices numbers above the table, so a
        // tbody-only refresh (like the Packages page uses) would leave those stale.
        window.location.reload();
      } else {
        Swal.fire({
          icon: "error",
          title: "Error",
          text: data.message || "Could not create the invoice.",
        });
        invoiceSubmitBtn.disabled = false;
      }
    } catch (error) {
      console.error("Error creating invoice:", error);
      Swal.fire({
        icon: "error",
        title: "Error",
        text: "Could not create the invoice. Please try again.",
      });
      invoiceSubmitBtn.disabled = false;
    }
  });

  /* ---------- Mark Paid modal ---------- */

  const markPaidModal = document.getElementById("markPaidModal");
  const markPaidForm = document.getElementById("markPaidForm");
  const markPaidSubmitBtn = document.getElementById("markPaidSubmitBtn");
  const markPaidInvoiceNumber = document.getElementById(
    "markPaidInvoiceNumber",
  );
  let markPaidInvoiceId = "";

  function openMarkPaidModal(id, invoiceNumber) {
    markPaidForm.reset();
    markPaidInvoiceId = id;
    markPaidInvoiceNumber.textContent = invoiceNumber;
    markPaidModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
    document.getElementById("transactionId").focus();
  }

  function closeMarkPaidModal() {
    markPaidModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  document.addEventListener("click", (e) => {
    const btn = e.target.closest(".mark-paid-btn");
    if (btn) openMarkPaidModal(btn.dataset.id, btn.dataset.invoiceNumber);
  });

  markPaidForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    markPaidSubmitBtn.disabled = true;

    try {
      const formData = new FormData(markPaidForm);
      formData.set("invoice_id", markPaidInvoiceId);

      const response = await fetch(INVOICE_ENDPOINT, {
        method: "PUT",
        body: formData,
      });

      const data = await response.json();

      if (data.status === "ok") {
        window.location.reload();
      } else {
        Swal.fire({
          icon: "error",
          title: "Error",
          text: data.message || "Could not mark the invoice paid.",
        });
        markPaidSubmitBtn.disabled = false;
      }
    } catch (error) {
      console.error("Error marking invoice paid:", error);
      Swal.fire({
        icon: "error",
        title: "Error",
        text: "Could not mark the invoice paid. Please try again.",
      });
      markPaidSubmitBtn.disabled = false;
    }
  });

    /* ---------- Invoice Details modal ---------- */

  const invoiceDetailsModal = document.getElementById("invoiceDetailsModal");

  function setInvoiceDetailsText(id, value) {
    const element = document.getElementById(id);

    if (element) {
      element.textContent =
        value === null || value === undefined || value === ""
          ? "—"
          : String(value);
    }
  }

  function formatInvoiceDate(value) {
    if (!value) return "—";

    const date = new Date(value);

    if (Number.isNaN(date.getTime())) return "—";

    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    }).format(date);
  }

  function formatInvoiceAmount(value) {
    if (value === null || value === undefined || value === "") {
      return "—";
    }

    const amount = Number(value);

    if (!Number.isFinite(amount)) return "—";

    return new Intl.NumberFormat(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount);
  }

  function resetInvoiceDetailsModal() {
    setInvoiceDetailsText("invoiceDetailsInvoiceNumber", "—");
    setInvoiceDetailsText("invoiceDetailsNumber", "—");
    setInvoiceDetailsText("invoiceDetailsStatus", "—");
    setInvoiceDetailsText("invoiceDetailsInvoiceDate", "—");
    setInvoiceDetailsText("invoiceDetailsDueDate", "—");
    setInvoiceDetailsText("invoiceDetailsAmount", "—");
    setInvoiceDetailsText("invoiceDetailsTransactionId", "—");
    setInvoiceDetailsText("invoiceDetailsPaidDate", "—");
    setInvoiceDetailsText("invoiceDetailsStudentName", "—");
    setInvoiceDetailsText("invoiceDetailsStudentId", "—");
    setInvoiceDetailsText("invoiceDetailsEnrollmentRef", "—");
    setInvoiceDetailsText("invoiceDetailsEnrollmentId", "—");
    setInvoiceDetailsText("invoiceDetailsCourseName", "—");
    setInvoiceDetailsText("invoiceDetailsPackageName", "—");
  }

  function populateInvoiceDetails(data) {
    setInvoiceDetailsText(
      "invoiceDetailsInvoiceNumber",
      data.invoice_number,
    );
    setInvoiceDetailsText("invoiceDetailsNumber", data.invoice_number);
    setInvoiceDetailsText(
      "invoiceDetailsStatus",
      data.paid ? "Paid" : "Unpaid",
    );
    setInvoiceDetailsText(
      "invoiceDetailsInvoiceDate",
      formatInvoiceDate(data.invoice_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsDueDate",
      formatInvoiceDate(data.due_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsAmount",
      formatInvoiceAmount(data.amount),
    );
    setInvoiceDetailsText(
      "invoiceDetailsTransactionId",
      data.transaction_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsPaidDate",
      formatInvoiceDate(data.paid_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsStudentName",
      data.student_name,
    );
    setInvoiceDetailsText(
      "invoiceDetailsStudentId",
      data.student_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsEnrollmentRef",
      data.enrollment_reference,
    );
    setInvoiceDetailsText(
      "invoiceDetailsEnrollmentId",
      data.enrollment_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsCourseName",
      data.course_name,
    );
    setInvoiceDetailsText(
      "invoiceDetailsPackageName",
      data.package_name,
    );
  }

  function openInvoiceDetailsModal() {
    invoiceDetailsModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeInvoiceDetailsModal() {
    invoiceDetailsModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  async function loadInvoiceDetails(invoiceId) {
    resetInvoiceDetailsModal();
    openInvoiceDetailsModal();

    try {
      const response = await fetch(
        `/invoice/details?id=${encodeURIComponent(invoiceId)}`,
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

      if (!response.ok) {
        throw new Error(data.message || "Could not load invoice details.");
      }

      populateInvoiceDetails(data);
    } catch (error) {
      console.error("Error loading invoice details:", error);

      closeInvoiceDetailsModal();

      Swal.fire({
        icon: "error",
        title: "Could not load invoice",
        text:
          error.message ||
          "Could not load invoice details. Please try again.",
      });
    }
  }

  document.addEventListener("click", (e) => {
    const btn = e.target.closest(".view-invoice-btn");

    if (btn) {
      loadInvoiceDetails(btn.dataset.id);
    }

    if (e.target.closest("[data-close-invoice-details-modal]")) {
      closeInvoiceDetailsModal();
    }
  });

  /* ---------- Modal close wiring ---------- */

  invoiceModal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-invoice-modal]")) closeInvoiceModal();
  });
  markPaidModal.addEventListener("click", (e) => {
    if (e.target.closest("[data-close-markpaid-modal]")) closeMarkPaidModal();
  });
  document.addEventListener("keydown", (e) => {
    if (e.key !== "Escape") return;
    if (!invoiceModal.classList.contains("hidden")) closeInvoiceModal();
    if (!markPaidModal.classList.contains("hidden")) closeMarkPaidModal();
    if (!invoiceDetailsModal.classList.contains("hidden")) {
      closeInvoiceDetailsModal();
    }
  });
})();

  /* ---------- Invoice Details modal ---------- */

  const invoiceDetailsModal = document.getElementById("invoiceDetailsModal");

  function setInvoiceDetailsText(id, value) {
    const element = document.getElementById(id);

    if (element) {
      element.textContent =
        value === null || value === undefined || value === ""
          ? "—"
          : String(value);
    }
  }

  function formatInvoiceDate(value) {
    if (!value) return "—";

    const date = new Date(value);

    if (Number.isNaN(date.getTime())) return "—";

    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    }).format(date);
  }

  function formatInvoiceAmount(value) {
    if (value === null || value === undefined || value === "") {
      return "—";
    }

    const amount = Number(value);

    if (!Number.isFinite(amount)) return "—";

    return new Intl.NumberFormat(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount);
  }

  function resetInvoiceDetailsModal() {
    setInvoiceDetailsText("invoiceDetailsInvoiceNumber", "—");
    setInvoiceDetailsText("invoiceDetailsNumber", "—");
    setInvoiceDetailsText("invoiceDetailsStatus", "—");
    setInvoiceDetailsText("invoiceDetailsInvoiceDate", "—");
    setInvoiceDetailsText("invoiceDetailsDueDate", "—");
    setInvoiceDetailsText("invoiceDetailsAmount", "—");
    setInvoiceDetailsText("invoiceDetailsTransactionId", "—");
    setInvoiceDetailsText("invoiceDetailsPaidDate", "—");
    setInvoiceDetailsText("invoiceDetailsStudentName", "—");
    setInvoiceDetailsText("invoiceDetailsStudentId", "—");
    setInvoiceDetailsText("invoiceDetailsEnrollmentRef", "—");
    setInvoiceDetailsText("invoiceDetailsEnrollmentId", "—");
    setInvoiceDetailsText("invoiceDetailsCourseName", "—");
    setInvoiceDetailsText("invoiceDetailsPackageName", "—");
  }

  function populateInvoiceDetails(data) {
    setInvoiceDetailsText(
      "invoiceDetailsInvoiceNumber",
      data.invoice_number,
    );
    setInvoiceDetailsText("invoiceDetailsNumber", data.invoice_number);
    setInvoiceDetailsText(
      "invoiceDetailsStatus",
      data.paid ? "Paid" : "Unpaid",
    );
    setInvoiceDetailsText(
      "invoiceDetailsInvoiceDate",
      formatInvoiceDate(data.invoice_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsDueDate",
      formatInvoiceDate(data.due_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsAmount",
      formatInvoiceAmount(data.amount),
    );
    setInvoiceDetailsText(
      "invoiceDetailsTransactionId",
      data.transaction_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsPaidDate",
      formatInvoiceDate(data.paid_date),
    );
    setInvoiceDetailsText(
      "invoiceDetailsStudentName",
      data.student_name,
    );
    setInvoiceDetailsText(
      "invoiceDetailsStudentId",
      data.student_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsEnrollmentRef",
      data.enrollment_reference,
    );
    setInvoiceDetailsText(
      "invoiceDetailsEnrollmentId",
      data.enrollment_id,
    );
    setInvoiceDetailsText(
      "invoiceDetailsCourseName",
      data.course_name,
    );
    setInvoiceDetailsText(
      "invoiceDetailsPackageName",
      data.package_name,
    );
  }

  function openInvoiceDetailsModal() {
    invoiceDetailsModal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeInvoiceDetailsModal() {
    invoiceDetailsModal.classList.add("hidden");
    document.body.classList.remove("overflow-hidden");
  }

  async function loadInvoiceDetails(invoiceId) {
    resetInvoiceDetailsModal();
    openInvoiceDetailsModal();

    try {
      const response = await fetch(
        `/invoice/details?id=${encodeURIComponent(invoiceId)}`,
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

      if (!response.ok) {
        throw new Error(data.message || "Could not load invoice details.");
      }

      populateInvoiceDetails(data);
    } catch (error) {
      console.error("Error loading invoice details:", error);

      closeInvoiceDetailsModal();

      Swal.fire({
        icon: "error",
        title: "Could not load invoice",
        text:
          error.message ||
          "Could not load invoice details. Please try again.",
      });
    }
  }

  document.addEventListener("click", (e) => {
    const btn = e.target.closest(".view-invoice-btn");

    if (btn) {
      loadInvoiceDetails(btn.dataset.id);
    }

    if (e.target.closest("[data-close-invoice-details-modal]")) {
      closeInvoiceDetailsModal();
    }
    
  });
