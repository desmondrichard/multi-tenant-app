import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import { Link } from "react-router-dom";
import { CheckCircle2, Circle, Clock, LayoutList } from "lucide-react";
import Pagination from "../components/Pagination";

const PAGE_SIZE = 10;

export default function MyTasks() {
  const [page, setPage] = useState(1);

  // One page of the tasks assigned to me
  const { data, isLoading } = useQuery({
    queryKey: ["my-tasks", page],
    queryFn: async () => {
      const res = await api.get(`/my-tasks?page=${page}&limit=${PAGE_SIZE}`);
      return res.data;
    },
    placeholderData: (previous) => previous,
  });
  const tasks = data?.tasks;

  // All project names (small list) to show the project of each task
  const { data: projects } = useQuery({
    queryKey: ["projects"],
    queryFn: async () => {
      const res = await api.get("/projects");
      return res.data;
    },
  });

  // If the last task of a page disappeared, go back one page
  useEffect(() => {
    if (data && data.tasks.length === 0 && page > 1) {
      setPage((p) => p - 1);
    }
  }, [data, page]);

  if (isLoading)
    return <div className="p-8 text-slate-500">Loading your tasks...</div>;

  const getProjectName = (projectId) => {
    return projects?.find((p) => p.id === projectId)?.name || "Unknown Project";
  };

  return (
    <div className="max-w-5xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <LayoutList className="w-6 h-6 text-brand-600" /> My Tasks
        </h1>
        <p className="text-slate-500 mt-1">
          All tasks currently assigned to you across all projects.
        </p>
      </div>

      {tasks?.length === 0 ? (
        <div className="text-center py-20 bg-white rounded-xl border border-slate-200 border-dashed">
          <CheckCircle2 className="w-12 h-12 text-green-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-slate-900">
            You're all caught up!
          </h3>
          <p className="text-slate-500">
            You have no tasks assigned to you right now.
          </p>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-xl text-left border-collapse">
              <thead>
                <tr className="bg-slate-50 border-b border-slate-200 text-sm text-slate-600">
                  <th className="p-4 font-medium">Task</th>
                  <th className="p-4 font-medium">Project</th>
                  <th className="p-4 font-medium">Status</th>
                  <th className="p-4 font-medium">Priority</th>
                </tr>
              </thead>
              <tbody>
                {tasks?.map((task) => (
                  <tr
                    key={task.id}
                    className="border-b border-slate-100 last:border-0 hover:bg-slate-50 transition-colors"
                  >
                    <td className="p-4">
                      {/* ?task= opens the task details right away */}
                      <Link
                        to={`/projects/${task.projectId}?task=${task.id}`}
                        className="font-medium text-slate-900 hover:text-brand-600"
                      >
                        {task.title}
                      </Link>
                      <p className="text-xs text-slate-500 mt-1 line-clamp-1">
                        {task.description}
                      </p>
                    </td>
                    <td className="p-4 text-sm text-slate-600">
                      {getProjectName(task.projectId)}
                    </td>
                    <td className="p-4">
                      <span
                        className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium capitalize ${
                          task.status === "done"
                            ? "bg-green-100 text-green-700"
                            : task.status === "under_review"
                              ? "bg-amber-100 text-amber-700"
                              : task.status === "in_progress"
                                ? "bg-blue-100 text-blue-700"
                                : "bg-slate-100 text-slate-700"
                        }`}
                      >
                        {task.status === "done" ? (
                          <CheckCircle2 className="w-3.5 h-3.5" />
                        ) : task.status === "in_progress" ||
                          task.status === "under_review" ? (
                          <Clock className="w-3.5 h-3.5" />
                        ) : (
                          <Circle className="w-3.5 h-3.5" />
                        )}
                        {task.status.replace("_", " ")}
                      </span>
                    </td>
                    <td className="p-4">
                      <span
                        className={`text-[10px] font-bold uppercase px-2 py-1 rounded-md ${
                          task.priority === "high"
                            ? "bg-red-100 text-red-700"
                            : task.priority === "medium"
                              ? "bg-yellow-100 text-yellow-700"
                              : "bg-slate-100 text-slate-600"
                        }`}
                      >
                        {task.priority}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <Pagination
        page={page}
        limit={PAGE_SIZE}
        totalCount={data?.totalCount || 0}
        onPageChange={setPage}
        label="tasks"
      />
    </div>
  );
}
