// Toast for HTMX errors
document.body.addEventListener("htmx:responseError", e => {
    const toast = document.getElementById("toast");
    document.getElementById("toast-text").textContent =
        e.detail.xhr.responseText || "Something went wrong";
    toast.classList.remove("hidden");
    requestAnimationFrame(() => toast.classList.add("opacity-100"));
    setTimeout(() => toast.classList.remove("opacity-100"), 4000);
    setTimeout(() => toast.classList.add("hidden"), 4500);
});

// Sidebar active state
window.setActive = btn => {
    const activeStateClasses = ["bg-blue-50", "text-blue-700"]
    document.querySelectorAll(".nav-btn").forEach(
        b => b.classList.remove(...activeStateClasses)
    );
    btn.classList.add(...activeStateClasses);
};

// Autofocus textarea
document.body.addEventListener("htmx:afterSwap", e => {
    const input = e.target.querySelector?.("#url-input");
    if (input) input.focus();
});

// Submit for on ENTER key press
document.body.addEventListener("keydown", e => {
    if (e.isComposing) return;

    const t = e.target;
    if (t?.id === "url-input" && e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        t.form?.requestSubmit();
    }
});

// UTC → Local time
const convertUTCToLocal = () => {
    const el = document.getElementById("last-visited");
    if (!el?.dataset.utc) return;
    const d = new Date(el.dataset.utc);
    el.textContent = isNaN(d)
        ? "Never"
        : d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
};

// Refresh icons + timestamps
const refreshUI = () => {
    window.lucide?.createIcons();
    convertUTCToLocal();
};
["DOMContentLoaded", "load", "htmx:afterSwap"].forEach(evt =>
    document.addEventListener(evt, refreshUI)
);