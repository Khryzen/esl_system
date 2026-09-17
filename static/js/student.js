document.addEventListener("DOMContentLoaded", function () {
  const modal = document.getElementById("addStudentModal");
  const openButton = document.getElementById("openAddStudentModal");
  const closeButton = document.getElementById("closeAddStudentModal");
  const cancelButton = document.getElementById("cancelAddStudentModal");
  const backdrop = document.getElementById("addStudentBackdrop");

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
  addStudentForm.addEventListener('submit', function(e){
    e.preventDefault();

    const formData = new FormData(addStudentForm);
  })
});
