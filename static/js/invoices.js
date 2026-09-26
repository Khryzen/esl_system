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
        alert("Error: " + (data.message || "Could not create the invoice."));
        invoiceSubmitBtn.disabled = false;
      }
    } catch (error) {
      console.error("Error creating invoice:", error);
      alert("Could not create the invoice. Please try again.");
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
      const response = await fetch(
        `${INVOICE_ENDPOINT}?id=${encodeURIComponent(markPaidInvoiceId)}`,
        {
          method: "PUT",
          body: new FormData(markPaidForm),
        },
      );
      const data = await response.json();

      if (data.status === "ok") {
        window.location.reload(); // same reasoning as above: totals up top would go stale otherwise
      } else {
        alert("Error: " + (data.message || "Could not mark the invoice paid."));
        markPaidSubmitBtn.disabled = false;
      }
    } catch (error) {
      console.error("Error marking invoice paid:", error);
      alert("Could not mark the invoice paid. Please try again.");
      markPaidSubmitBtn.disabled = false;
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
  });
})();
