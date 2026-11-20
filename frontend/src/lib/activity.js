import { useEffect, useState } from "react";

const STATUS_LABELS = {
    todo: "To Do",
    in_progress: "In Progress",
    under_review: "Under Review",
    done: "Done",
};

// Turns one activity row from the API into a readable sentence + an icon type
export function describeActivity(a) {
    const who = a.userName || "Someone";
    const task = a.taskTitle ? `"${a.taskTitle}"` : "a task";

    switch (a.action) {
        case "created_task":
            return { type: "created", text: `${who} created task ${task}` };

        case "changed_status": {
            const raw = (a.details || "").replace(/^Changed status to\s*/i, "").trim();
            const label = STATUS_LABELS[raw] || raw || "a new status";
            return {
                type: raw === "done" ? "done" : "status",
                text: `${who} moved ${task} to ${label}`,
            };
        }

        case "added_comment": {
            const m = (a.details || "").match(/^Commented on subtask:\s*(.+)$/);
            return {
                type: "comment",
                text: m
                    ? `${who} commented on subtask "${m[1]}" in ${task}`
                    : `${who} commented on ${task}`,
            };
        }

        case "deleted_task":
            return { type: "deleted", text: `${who} deleted task ${task}` };

        default:
            return { type: "created", text: `${who} ${a.details || "updated"} ${task}` };
    }
}

// "just now", "5 minutes ago", "2 hours ago", "3 days ago", then a date
export function timeAgo(dateStr) {
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 45) return "just now";
    const minutes = Math.round(seconds / 60);
    if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
    const hours = Math.round(minutes / 60);
    if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
    const days = Math.round(hours / 24);
    if (days < 7) return `${days} day${days === 1 ? "" : "s"} ago`;
    return new Date(dateStr).toLocaleDateString();
}

// Where clicking an activity should go (null = nothing to open)
export function activityLink(a) {
    if (!a.projectId || a.taskDeleted || a.action === "deleted_task") return null;
    return `/projects/${a.projectId}?task=${a.taskId}`;
}

// Re-render every minute so "5 minutes ago" keeps updating
export function useMinuteTick() {
    const [, setTick] = useState(0);
    useEffect(() => {
        const id = setInterval(() => setTick((t) => t + 1), 60000);
        return () => clearInterval(id);
    }, []);
}