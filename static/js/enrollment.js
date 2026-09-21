function toggleStudentMode() {
  const isNew =
    document.querySelector('input[name="student_type"]:checked').value ===
    "new";
  const existingSection = document.getElementById("existing-student-section");
  const newSection = document.getElementById("new-student-section");

  existingSection.classList.toggle("hidden", isNew);
  existingSection.classList.toggle("block", !isNew);
  newSection.classList.toggle("hidden", !isNew);
  newSection.classList.toggle("grid", isNew);

  document.getElementById("StudentID").required = !isNew;
  newSection.querySelectorAll("input").forEach((input) => {
    input.required = isNew;
  });
}

function updateTotalClasses() {
  const packageSelect = document.getElementById("PackageID");
  const selectedOption = packageSelect.options[packageSelect.selectedIndex];
  document.getElementById("TotalClasses").value =
    selectedOption?.dataset.total ?? "";
}

function credsToText(creds) {
  if (typeof creds !== "object") return String(creds);
  return Object.entries(creds)
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n");
}

function showEnrollmentResult(data) {
  const panel = document.getElementById("enrollmentResult");
  document.getElementById("enrollmentRef").textContent = data.reference_number;

  const credsBox = document.getElementById("enrollmentCredsBox");
  credsBox.classList.toggle("hidden", !data.creds);
  if (data.creds) {
    document.getElementById("enrollmentCreds").textContent = credsToText(
      data.creds,
    );
  }

  panel.classList.remove("hidden");
  panel.scrollIntoView({ behavior: "smooth", block: "start" });
}

async function submitEnrollment(e) {
  e.preventDefault();

  const form = e.currentTarget;
  const submitBtn = form.querySelector('button[type="submit"]');
  const formData = new FormData(form);
  const isNewStudent = formData.get("student_type") === "new";
  const file = formData.get("Contract");
  if (file instanceof File && file.size === 0) formData.delete("Contract");

  submitBtn.disabled = true;
  try {
    const response = await fetch(window.location.pathname, {
      method: "POST",
      body: formData,
    });
    const data = await response.json();

    if (data.status !== "ok") {
      alert("Error: " + (data.message || "Something went wrong."));
      return;
    }
    if (isNewStudent) {
      const name = `${formData.get("NewStudentFirstName")} ${formData.get("NewStudentLastName")}`;
      document
        .getElementById("StudentID")
        .add(new Option(name, data.student_id));
    }

    form.reset();
    toggleStudentMode();
    updateTotalClasses();
    showEnrollmentResult(data);
  } catch (error) {
    console.error("Error saving enrollment:", error);
    alert("Could not save the enrollment. Please try again.");
  } finally {
    submitBtn.disabled = false;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  toggleStudentMode();
  updateTotalClasses();

  document
    .getElementById("enrollmentForm")
    .addEventListener("submit", submitEnrollment);
});
