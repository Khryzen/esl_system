const loginForm = document.getElementById("loginForm");
const passwordInput = document.getElementById("password");
const togglePassword = document.getElementById("togglePassword");
const eyeIcon = document.getElementById("eyeIcon");

const loginButton = document.getElementById("loginButton");
const loginText = document.getElementById("loginText");
const loadingIcon = document.getElementById("loadingIcon");
const loginError = document.getElementById("loginError");

// Toggle password visibility
togglePassword.addEventListener("click", () => {
  const isPassword = passwordInput.type === "password";

  passwordInput.type = isPassword ? "text" : "password";

  togglePassword.setAttribute(
    "aria-label",
    isPassword ? "Hide password" : "Show password",
  );

  eyeIcon.innerHTML = isPassword
    ? `<path d="M3.6 3.6 20.4 20.4" />
  <path d="M10.58 10.58a2 2 0 0 0 2.83 2.83" />
  <path d="M9.88 4.24A10.94 10.94 0 0 1 12 4c5 0 8.73 3.11 10 8a10.9 10.9 0 0 1-2.05 3.84" />
  <path d="M6.61 6.61C4.62 7.83 3.17 9.67 2 12c1.27 4.89 5 8 10 8a10.9 10.9 0 0 0 5.39-1.42" />`
    : `<path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" />
  <circle cx="12" cy="12" r="3" />`;
});

// Login
loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  loginError.classList.add("hidden");
  loginButton.disabled = true;
  loginText.textContent = "Signing in...";
  loadingIcon.classList.remove("hidden");

  try {
    const formData = new FormData(loginForm);

    const response = await fetch("/login/", {
      method: "POST",
      body: formData,
    });

    const result = await response.json();

    if (!response.ok) {
      loginError.textContent =
        result.message || "Invalid username or password.";
      loginError.classList.remove("hidden");
      return;
    }

    if (result.success && result.redirect) {
      window.location.href = result.redirect;
      return;
    }
  } catch (error) {
    console.error("Login error:", error);

    loginError.textContent = "Unable to sign in. Please try again.";
    loginError.classList.remove("hidden");
  } finally {
    loginButton.disabled = false;
    loginText.textContent = "Sign In";
    loadingIcon.classList.add("hidden");
  }
});