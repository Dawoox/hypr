const badge = document.getElementById("server-status-badge");
const line = document.getElementById("server-stats-line");

async function loadServerStats() {
    try {
        const resp = await fetch("/api/domain/stats");
        if (!resp.ok) {
            throw new Error("HTTP " + resp.status);
        }
        const stats = await resp.json();
        const size = stats.total_bandwidth >= 1024
            ? (stats.total_bandwidth / 1024).toFixed(1) + "GB"
            : stats.total_bandwidth + "MB";
        line.textContent = "Serving " + size + " monthly to " + stats.domain_count + " domains!";
        badge.textContent = "[ Online ]";
    } catch (e) {
        badge.textContent = "[ Offline ]";
        line.textContent = "Server stats unavailable";
    }
}

loadServerStats();
