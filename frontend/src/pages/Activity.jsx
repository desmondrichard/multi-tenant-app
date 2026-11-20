import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import {
  describeActivity,
  timeAgo,
  activityLink,
  useMinuteTick,
} from "../lib/activity";
import ActivityIcon from "../components/ActivityIcon";
import { ArrowLeft, ChevronLeft, ChevronRight, Zap } from "lucide-react";

const FILTERS = [
  { id: "all", label: "All" },
  { id: "created_task", label: "Created" },
  { id: "changed_status", label: "Status changes" },
  { id: "added_comment", label: "Comments" },
  { id: "deleted_task", label: "Deleted" },
];

const LIMIT = 15;

export default function Activity() {
  const navigate = useNavigate();
  const [action, setAction] = useState("all");
  const [userId, setUserId] = useState("all");
  const [page, setPage] = useState(1);
  useMinuteTick();

  // Live: the WebSocket handler invalidates ["activity-feed"] on every change
  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["activity-feed", "page", action, userId, page],
    queryFn: async () => {
      const params = new URLSearchParams({
        action,
        userId,
        page: String(page),
        limit: String(LIMIT),
      });
      const res = await api.get(`/activities?${params.toString()}`);
      return res.data;
    },
    placeholderData: (previous) => previous,
    refetchInterval: 60000, // safety net if the WebSocket is down
  });

  const { data: members } = useQuery({
    queryKey: ["users"],
    queryFn: async () => {
      const res = await api.get("/users");
      return res.data;
    },
  });

  const activities = data?.activities || [];
  const totalCount = data?.totalCount || 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / LIMIT));

  const changeAction = (id) => {
    setAction(id);
    setPage(1);
  };
  const changeUser = (id) => {
    setUserId(id);
    setPage(1);
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <button
          onClick={() => navigate(-1)}
          className="flex items-center gap-1 text-sm text-slate-500 hover:text-slate-900 mb-3"
        >
          <ArrowLeft className="w-4 h-4" /> Back
        </button>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
              <Zap className="w-6 h-6 text-brand-500" /> Activity
            </h1>
            <p className="text-slate-500 mt-1">
              Everything happening in your workspace, updated live.
            </p>
          </div>
          <span className="flex items-center gap-2 text-xs font-medium text-green-700 bg-green-50 border border-green-100 px-3 py-1 rounded-full">
            <span
              className={`w-2 h-2 rounded-full bg-green-500 ${isFetching ? "animate-pulse" : ""}`}
            />
            Live
          </span>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <div className="flex flex-wrap gap-2">
          {FILTERS.map((f) => (
            <button
              key={f.id}
              onClick={() => changeAction(f.id)}
              className={`px-3 py-1.5 rounded-full text-sm font-medium border transition-colors ${
                action === f.id
                  ? "bg-brand-600 text-white border-brand-600"
                  : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>

        <select
          value={userId}
          onChange={(e) => changeUser(e.target.value)}
          className="ml-auto px-3 py-1.5 text-sm border border-slate-300 rounded-lg bg-white focus:ring-2 focus:ring-brand-500"
        >
          <option value="all">All members</option>
          {members?.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </div>

      {/* List */}
      <div className="bg-white rounded-2xl border border-slate-100 soft-shadow">
        {isLoading ? (
          <p className="p-8 text-center text-slate-500">Loading activity...</p>
        ) : activities.length === 0 ? (
          <p className="p-8 text-center text-slate-500">
            No activity found for these filters.
          </p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {activities.map((activity) => {
              const { type, text } = describeActivity(activity);
              const link = activityLink(activity);
              const row = (
                <div className="flex items-start gap-4 p-4 hover:bg-slate-50 transition-colors">
                  <ActivityIcon type={type} className="mt-0.5 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-800">{text}</p>
                    {activity.taskDeleted &&
                      activity.action !== "deleted_task" && (
                        <p className="text-xs text-slate-400 mt-0.5">
                          This task has since been deleted
                        </p>
                      )}
                  </div>
                  <span
                    className="text-xs text-slate-400 shrink-0"
                    title={new Date(activity.createdAt).toLocaleString()}
                  >
                    {timeAgo(activity.createdAt)}
                  </span>
                </div>
              );
              return (
                <li key={activity.id}>
                  {link ? <Link to={link}>{row}</Link> : row}
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* Pagination */}
      {totalCount > LIMIT && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-sm text-slate-500">
            Page {page} of {totalPages} · {totalCount} activities
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="p-2 border border-slate-200 rounded-lg bg-white hover:bg-slate-50 disabled:opacity-40"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="p-2 border border-slate-200 rounded-lg bg-white hover:bg-slate-50 disabled:opacity-40"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
