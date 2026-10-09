(() => {
  "use strict";

  const INVOICE_ENDPOINT = window.location.pathname;
  const PAYMENT_ENDPOINT = "/admin/invoice/payment";
  const DETAILS_ENDPOINT = "/admin/invoice/details";
  const PAYMENT_HISTORY_ENDPOINT = "/admin/invoice/payment-history";

  const $ = (id) => document.getElementById(id);

  function showError(title, message) {
    Swal.fire({
      icon: "error",
      title,
      text: message,
    });
  }

  function showSuccess(title, message) {
    return Swal.fire({
      icon: "success",
      title,
      text: message,
    });
  }

async function readJSON(response) {
  const contentType = response.headers.get("content-type") || "";
  const body = await response.text();

  let data;

  if (contentType.includes("application/json")) {
    try {
      data = JSON.parse(body);
    } catch {
      throw new Error(`Invalid JSON response (HTTP ${response.status}).`);
    }
  } else {
    const message = body
      .trim()
      .replace(/<[^>]*>/g, " ")
      .trim();

    throw new Error(message || `Unexpected response from server (HTTP ${response.status}).`);
  }

  if (!response.ok) {
    throw new Error(data.message || `Server returned HTTP ${response.status}.`);
  }

  if (data.status && data.status !== "ok") {
    throw new Error(data.message || "The request could not be completed.");
  }

  return data;
}

  function formatAmount(value) {
    const amount = Number(value);

    if (!Number.isFinite(amount)) {
      return "—";
    }

    return new Intl.NumberFormat(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount);
  }

  function formatDate(value) {
    if (!value) {
      return "—";
    }

    const date = new Date(value);

    if (Number.isNaN(date.getTime())) {
      return String(value);
    }

    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    }).format(date);
  }

  function setText(id, value) {
    const element = $(id);

    if (!element) {
      return;
    }

    element.textContent = value === null || value === undefined || value === "" ? "—" : String(value);
  }

  function openModal(modal) {
    if (!modal) {
      return;
    }

    modal.classList.remove("hidden");
    document.body.classList.add("overflow-hidden");
  }

  function closeModal(modal) {
    if (!modal) {
      return;
    }

    modal.classList.add("hidden");

    const anyModalOpen = [$("invoiceModal"), $("recordPaymentModal"), $("invoiceDetailsModal")].some(
      (item) => item && !item.classList.contains("hidden"),
    );

    if (!anyModalOpen) {
      document.body.classList.remove("overflow-hidden");
    }
  }

  /* ---------- Invoice filters ---------- */

  const filterButtons = document.querySelectorAll("[data-invoice-filter]");

  function applyFilter(activeButton) {
    const activeClasses = "bg-gray-900 text-white dark:bg-white dark:text-gray-900";
    const inactiveClasses = "text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800";

    filterButtons.forEach((button) => {
      button.className =
        "invoice-filter-btn rounded-md px-3 py-1.5 text-sm font-medium " +
        (button === activeButton ? activeClasses : inactiveClasses);
    });

    const filter = activeButton.dataset.invoiceFilter;

    document.querySelectorAll("#invoicesTable tbody tr[data-paid]").forEach((row) => {
      const paid = row.dataset.paid === "true";

      const visible = filter === "all" || (filter === "unpaid" && !paid) || (filter === "paid" && paid);

      row.classList.toggle("hidden", !visible);
    });
  }

  filterButtons.forEach((button) => {
    button.addEventListener("click", () => applyFilter(button));
  });

  const allFilter = document.querySelector('[data-invoice-filter="all"]');

  if (allFilter) {
    applyFilter(allFilter);
  }

  /* ---------- Add Invoice modal ---------- */

  const invoiceModal = $("invoiceModal");
  const invoiceForm = $("invoiceForm");
  const invoiceSubmitButton = $("invoiceSubmitBtn");
  const enrollmentSelect = $("invoiceEnrollment");
  const enrollmentEmpty = $("invoiceEnrollmentEmpty");
  const amountInput = $("invoiceAmount");
  const dueDateInput = $("invoiceDueDate");

  function openInvoiceModal() {
    invoiceForm.reset();

    const hasEnrollments = enrollmentSelect.options.length > 1;

    enrollmentEmpty.classList.toggle("hidden", hasEnrollments);
    enrollmentSelect.disabled = !hasEnrollments;
    invoiceSubmitButton.disabled = !hasEnrollments;

    const dueDate = new Date();
    dueDate.setDate(dueDate.getDate() + 14);

    const year = dueDate.getFullYear();
    const month = String(dueDate.getMonth() + 1).padStart(2, "0");
    const day = String(dueDate.getDate()).padStart(2, "0");

    dueDateInput.value = `${year}-${month}-${day}`;

    openModal(invoiceModal);
    enrollmentSelect.focus();
  }

  function closeInvoiceModal() {
    closeModal(invoiceModal);
  }

  $("openAddInvoiceModal").addEventListener("click", openInvoiceModal);

  enrollmentSelect.addEventListener("change", () => {
    const selectedOption = enrollmentSelect.selectedOptions[0];
    const price = selectedOption?.dataset.price;

    if (price) {
      amountInput.value = price;
    }
  });

  invoiceForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    invoiceSubmitButton.disabled = true;

    try {
      const response = await fetch(INVOICE_ENDPOINT, {
        method: "POST",
        body: new FormData(invoiceForm),
        headers: {
          Accept: "application/json",
        },
      });

      const data = await readJSON(response);

      if (data.status === "ok") {
        await showSuccess("Invoice created", "The invoice was created successfully.");
        window.location.reload();
      }
    } catch (error) {
      console.error("Error creating invoice:", error);
      showError("Could not create invoice", error.message);
      invoiceSubmitButton.disabled = false;
    }
  });

  /* ---------- Record Payment modal ---------- */

  const recordPaymentModal = $("recordPaymentModal");
  const recordPaymentForm = $("recordPaymentForm");
  const recordPaymentSubmitButton = $("recordPaymentSubmitBtn");
  const invoiceIdInput = $("recordPaymentInvoiceId");

  function openRecordPaymentModal(button) {
    if (!recordPaymentForm || !recordPaymentModal || !invoiceIdInput) {
      showError(
        "Payment form error",
        "The Record Payment form is missing required elements.",
      );
      return;
    }

    recordPaymentForm.reset();

    const invoiceId = String(button.dataset.id || "").trim();
    const invoiceNumber = button.dataset.invoiceNumber || "Invoice";
    const invoiceAmount = button.dataset.amount;

    if (!/^[1-9]\d*$/.test(invoiceId)) {
      console.error("Invalid invoice ID on Record Payment button:", {
        dataset: { ...button.dataset },
        invoiceId,
      });

      showError(
        "Invalid invoice ID",
        "The selected invoice does not have a valid ID. Check the invoice button in the template.",
      );
      return;
    }

    invoiceIdInput.value = invoiceId;
    console.log("Invoice ID: ", invoiceId);

    $("recordPaymentInvoiceNumber").textContent = invoiceNumber;
    $("recordPaymentInvoiceAmount").textContent =
      formatAmount(invoiceAmount);
    $("recordPaymentAmount").value = "";

    openModal(recordPaymentModal);
    $("recordPaymentAmount").focus();
  }

  function closeRecordPaymentModal() {
    closeModal(recordPaymentModal);
  }

  document.addEventListener("click", (event) => {
    const button = event.target.closest(".record-payment-btn");

    if (button) {
      event.preventDefault();
      openRecordPaymentModal(button);
      return;
    }

    if (event.target.closest("[data-close-record-payment]")) {
      closeRecordPaymentModal();
    }
  });

  if (recordPaymentForm && recordPaymentSubmitButton) {
    recordPaymentForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      if (recordPaymentSubmitButton.disabled) {
        return;
      }

      const invoiceId = String(invoiceIdInput?.value || "").trim();

      if (!/^[1-9]\d*$/.test(invoiceId)) {
        showError(
          "Invalid invoice ID",
          "The invoice ID is missing or invalid. Close this form and select the invoice again.",
        );
        return;
      }

      if (!recordPaymentForm.reportValidity()) {
        return;
      }

      recordPaymentSubmitButton.disabled = true;

      try {
        const formData = new FormData(recordPaymentForm);

        // Ensure the server receives the validated invoice ID.
        formData.set("invoice_id", invoiceId);

        console.log("Submitting invoice payment:", {
          invoice_id: formData.get("invoice_id"),
          amount: formData.get("amount"),
          payment_method: formData.get("payment_method"),
        });

        const response = await fetch(PAYMENT_ENDPOINT, {
          method: "POST",
          body: formData,
          headers: {
            Accept: "application/json",
          },
        });

        await readJSON(response);

        closeRecordPaymentModal();

        await showSuccess(
          "Payment recorded",
          "The payment has been recorded successfully.",
        );

        window.location.reload();
      } catch (error) {
        console.error("Error recording payment:", error);

        await showError(
          "Could not record payment",
          error.message || "Please try again.",
        );
      } finally {
        recordPaymentSubmitButton.disabled = false;
      }
    });
  }


  /* ---------- Invoice details ---------- */

  const invoiceDetailsModal = $("invoiceDetailsModal");

  function resetInvoiceDetails() {
    [
      "invoiceDetailsInvoiceNumber",
      "invoiceDetailsNumber",
      "invoiceDetailsStatus",
      "invoiceDetailsInvoiceDate",
      "invoiceDetailsDueDate",
      "invoiceDetailsAmount",
      "invoiceDetailsPaidDate",
      "invoiceDetailsStudentName",
      "invoiceDetailsStudentId",
      "invoiceDetailsEnrollmentRef",
      "invoiceDetailsEnrollmentId",
      "invoiceDetailsCourseName",
      "invoiceDetailsPackageName",
      "invoiceDetailsTotalPaid",
      "invoiceDetailsBalance",
      "invoiceDetailsPaymentStatus",
    ].forEach((id) => setText(id, "—"));

    $("invoicePaymentHistory").replaceChildren();

    const loading = document.createElement("p");
    loading.className = "text-sm text-gray-500";
    loading.textContent = "Loading payment history…";
    $("invoicePaymentHistory").appendChild(loading);
  }

  function populateInvoiceDetails(data) {
    setText("invoiceDetailsInvoiceNumber", data.invoice_number);
    setText("invoiceDetailsNumber", data.invoice_number);
    setText("invoiceDetailsStatus", data.paid ? "Paid" : "Outstanding");
    setText("invoiceDetailsInvoiceDate", formatDate(data.invoice_date));
    setText("invoiceDetailsDueDate", formatDate(data.due_date));
    setText("invoiceDetailsAmount", formatAmount(data.amount));
    setText("invoiceDetailsPaidDate", formatDate(data.paid_date));

    setText("invoiceDetailsStudentName", data.student_name);
    setText("invoiceDetailsStudentId", data.student_id);
    setText("invoiceDetailsEnrollmentRef", data.enrollment_reference);
    setText("invoiceDetailsEnrollmentId", data.enrollment_id);
    setText("invoiceDetailsCourseName", data.course_name);
    setText("invoiceDetailsPackageName", data.package_name);

    setText("invoiceDetailsTotalPaid", formatAmount(data.total_paid));
    setText("invoiceDetailsBalance", formatAmount(data.balance));
    setText("invoiceDetailsPaymentStatus", data.payment_status || (data.paid ? "Paid" : "Unpaid"));
  }

  function renderPaymentHistory(payments) {
    const container = $("invoicePaymentHistory");
    container.replaceChildren();

    if (!Array.isArray(payments) || payments.length === 0) {
      const empty = document.createElement("p");
      empty.className = "text-sm text-gray-500";
      empty.textContent = "No payments have been recorded for this invoice.";
      container.appendChild(empty);
      return;
    }

    payments.forEach((payment) => {
      const card = document.createElement("div");
      card.className = "rounded-xl border border-gray-200 p-4 dark:border-gray-700";

      const header = document.createElement("div");
      header.className = "flex flex-wrap items-start justify-between gap-3";

      const methodGroup = document.createElement("div");

      const method = document.createElement("p");
      method.className = "text-sm font-semibold text-gray-900 dark:text-white";
      method.textContent = payment.payment_method || "Payment";

      const date = document.createElement("p");
      date.className = "mt-1 text-xs text-gray-500 dark:text-gray-400";
      date.textContent = formatDate(payment.payment_date);

      methodGroup.append(method, date);

      const amount = document.createElement("p");
      amount.className = "text-sm font-semibold text-gray-900 dark:text-white";
      amount.textContent = formatAmount(payment.amount);

      header.append(methodGroup, amount);
      card.appendChild(header);

      if (payment.reference_number) {
        const reference = document.createElement("p");
        reference.className = "mt-3 text-xs text-gray-500 dark:text-gray-400";
        reference.textContent = `Reference: ${payment.reference_number}`;
        card.appendChild(reference);
      }

      if (payment.notes) {
        const notes = document.createElement("p");
        notes.className = "mt-2 whitespace-pre-wrap text-sm text-gray-600 dark:text-gray-300";
        notes.textContent = payment.notes;
        card.appendChild(notes);
      }

      container.appendChild(card);
    });
  }

  async function loadPaymentHistory(invoiceId) {
    try {
      const response = await fetch(`${PAYMENT_HISTORY_ENDPOINT}?id=${encodeURIComponent(invoiceId)}`, {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
      });

      const data = await readJSON(response);
      renderPaymentHistory(data.payments);
    } catch (error) {
      console.error("Error loading payment history:", error);

      const container = $("invoicePaymentHistory");
      container.replaceChildren();

      const message = document.createElement("p");
      message.className = "text-sm text-red-600";
      message.textContent = error.message || "Could not load payment history.";
      container.appendChild(message);
    }
  }

  async function loadInvoiceDetails(invoiceId) {
    resetInvoiceDetails();
    openModal(invoiceDetailsModal);

    try {
      const response = await fetch(`${DETAILS_ENDPOINT}?id=${encodeURIComponent(invoiceId)}`, {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
      });

      const data = await readJSON(response);

      if (!data || !data.id) {
        throw new Error("The server response does not contain invoice details.");
      }

      populateInvoiceDetails(data);

      await loadPaymentHistory(invoiceId);
    } catch (error) {
      console.error("Error loading invoice details:", error);

      closeModal(invoiceDetailsModal);

      showError("Could not load invoice", error.message || "Please try again.");
    }
  }

  document.addEventListener("click", (event) => {
    const button = event.target.closest(".view-invoice-btn");

    if (button) {
      loadInvoiceDetails(button.dataset.id);
      return;
    }

    if (event.target.closest("[data-close-invoice-details-modal]")) {
      closeModal(invoiceDetailsModal);
    }
  });

  /* ---------- Modal close wiring ---------- */

  invoiceModal.addEventListener("click", (event) => {
    if (event.target.closest("[data-close-invoice-modal]")) {
      closeInvoiceModal();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }

    if (!invoiceModal.classList.contains("hidden")) {
      closeInvoiceModal();
    }

    if (!recordPaymentModal.classList.contains("hidden")) {
      closeRecordPaymentModal();
    }

    if (!invoiceDetailsModal.classList.contains("hidden")) {
      closeModal(invoiceDetailsModal);
    }
  });
})();
