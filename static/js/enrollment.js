function toggleStudentMode() {
  const mode = document.querySelector(
    'input[name="student_type"]:checked',
  ).value;
  const existingSection = document.getElementById("existing-student-section");
  const newSection = document.getElementById("new-student-section");

  if (mode === "new") {
    existingSection.classList.add("hidden");
    existingSection.classList.remove("block");
    newSection.classList.add("grid");
    newSection.classList.remove("hidden");
  } else {
    existingSection.classList.add("block");
    existingSection.classList.remove("hidden");
    newSection.classList.add("hidden");
    newSection.classList.remove("grid");
  }
}

function updateTotalClasses(){
  const packageSelect = document.getElementById('PackageID');
  const selectedOption = packageSelect.options[packageSelect.selectedOption];
  
  const totalClassInput = document.getElementById('TotalClasses');
  totalClassInput.value = selectedOption.dataset.total;
}