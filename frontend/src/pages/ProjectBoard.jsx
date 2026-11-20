import { useState, useEffect } from "react";
import {
  useParams,
  Link,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useSelector } from "react-redux";
import { api } from "../lib/api";
import { toast } from "sonner";
import {
  ArrowLeft,
  Plus,
  MoreHorizontal,
  Search,
  ChevronLeft,
  ChevronRight,
  Clock,
  Sparkles,
  Trash2,
} from "lucide-react";
import TaskDrawer from "../components/TaskDrawer";

const COLUMNS = [
  { id: "todo", title: "To Do" },
  { id: "in_progress", title: "In Progress" },
  { id: "under_review", title: "Under Review" },
  { id: "done", title: "Done" },
];

export default function ProjectBoard() {
  const { projectId } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { user } = useSelector((state) => state.auth);

  // Standard Modal States
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState(null);
  const [formData, setFormData] = useState({
    title: "",
    description: "",
    priority: "medium",
    assigneeId: "",
    dueDate: "",
  });

  // AI Modal States
  const [isAIModalOpen, setIsAIModalOpen] = useState(false);
  const [aiPrompt, setAiPrompt] = useState("");

  // Search, Filter, Pagination States
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [filterPriority, setFilterPriority] = useState("");
  const [sortBy, setSortBy] = useState("newest");
  const [page, setPage] = useState(1);
  const limit = 10;

  // Debounce Effect
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 500);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch Tasks
  const {
    data: response,
    isLoading,
    isFetching,
  } = useQuery({
    queryKey: [
      "tasks",
      projectId,
      debouncedSearch,
      filterStatus,
      filterPriority,
      sortBy,
      page,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        search: debouncedSearch,
        status: filterStatus,
        priority: filterPriority,
        sort: sortBy,
        page: page.toString(),
        limit: limit.toString(),
      });
      const res = await api.get(
        `/projects/${projectId}/tasks?${params.toString()}`,
      );
      return res.data;
    },
    placeholderData: (previousData) => previousData,
  });

  const tasks = response?.tasks || [];
  const totalCount = response?.totalCount || 0;
  const totalPages = Math.ceil(totalCount / limit);

  // Fetch Members
  const { data: members } = useQuery({
    queryKey: ["users"],
    queryFn: async () => {
      const res = await api.get("/users");
      return res.data;
    },
  });

  // Open a task directly from a notification / activity link: /projects/:id?task=TASK_ID
  const [searchParams, setSearchParams] = useSearchParams();
  useEffect(() => {
    const taskId = searchParams.get("task");
    if (!taskId) return;
    api
      .get(`/projects/${projectId}/tasks/${taskId}`)
      .then((res) => setSelectedTask(res.data))
      .catch(() => toast.error("That task no longer exists"))
      .finally(() => setSearchParams({}, { replace: true }));
  }, [searchParams, projectId, setSearchParams]);

  // Mutations
  const createMutation = useMutation({
    mutationFn: (newTask) => api.post(`/projects/${projectId}/tasks`, newTask),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
      toast.success("Task created!");
      setIsModalOpen(false);
      setFormData({
        title: "",
        description: "",
        priority: "medium",
        assigneeId: "",
        dueDate: "",
      });
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to create task"),
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ taskId, status }) =>
      api.patch(`/projects/${projectId}/tasks/${taskId}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
      toast.success("Task moved!");
    },
    onError: () => toast.error("Failed to move task"),
  });

  // Soft delete a task (hidden, but kept in the database)
  const deleteTaskMutation = useMutation({
    mutationFn: (taskId) =>
      api.delete(`/projects/${projectId}/tasks/${taskId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
      queryClient.invalidateQueries({ queryKey: ["my-tasks"] });
      toast.success("Task deleted");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to delete task"),
  });

  // Soft delete this whole project (and its tasks)
  const deleteProjectMutation = useMutation({
    mutationFn: () => api.delete(`/projects/${projectId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
      queryClient.invalidateQueries({ queryKey: ["my-tasks"] });
      toast.success("Project deleted");
      navigate("/projects");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to delete project"),
  });

  const handleDeleteProject = () => {
    if (
      window.confirm(
        "Delete this project and all of its tasks? Members will no longer see it.",
      )
    ) {
      deleteProjectMutation.mutate();
    }
  };

  // AI Mutation
  const aiMutation = useMutation({
    mutationFn: (prompt) =>
      api.post("/ai/generate-task", { prompt, projectId }),
    onSuccess: (res) => {
      const created = res.data;
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
      queryClient.invalidateQueries({ queryKey: ["activity-feed"] });

      const count = created.subtasks?.length || 0;
      toast.success(
        `AI created "${created.title}" with ${count} subtask${count === 1 ? "" : "s"}`,
      );
      setIsAIModalOpen(false);
      setAiPrompt("");
      setSelectedTask(created); // open it so you can review everything
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "AI failed to generate task"),
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    createMutation.mutate(formData);
  };

  if (isLoading && !response)
    return <div className="p-8 text-slate-500">Loading board...</div>;

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex flex-wrap justify-between items-center gap-3 mb-6">
        <div className="flex items-center gap-4">
          <Link
            to="/projects"
            className="p-2 hover:bg-slate-200 rounded-lg transition-colors"
          >
            <ArrowLeft className="w-5 h-5 text-slate-600" />
          </Link>
          <h1 className="text-2xl font-bold text-slate-900">Project Board</h1>
        </div>

        <div className="flex flex-wrap gap-3">
          <button
            onClick={handleDeleteProject}
            disabled={deleteProjectMutation.isPending}
            className="flex items-center gap-2 bg-white text-red-600 border border-red-200 px-4 py-2 rounded-lg text-sm font-medium hover:bg-red-50 transition-colors disabled:opacity-50"
          >
            <Trash2 className="w-4 h-4" /> Delete Project
          </button>

          <button
            onClick={() => setIsAIModalOpen(true)}
            className="flex items-center gap-2 bg-purple-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-purple-700 transition-colors"
          >
            <Sparkles className="w-4 h-4" /> AI Magic
          </button>

          <button
            onClick={() => setIsModalOpen(true)}
            className="flex items-center gap-2 bg-brand-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-brand-700 transition-colors"
          >
            <Plus className="w-4 h-4" /> New Task
          </button>
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap gap-4 mb-6 bg-white p-3 rounded-xl border border-slate-200 shadow-sm relative">
        <div className="flex-1 relative min-w-50">
          <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
          <input
            type="text"
            placeholder="Search tasks by title..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>

        <select
          value={filterStatus}
          onChange={(e) => {
            setFilterStatus(e.target.value);
            setPage(1);
          }}
          className="px-3 py-2 border border-slate-200 rounded-lg text-sm text-slate-600 focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          <option value="">All Categories</option>
          <option value="todo">To Do</option>
          <option value="in_progress">In Progress</option>
          <option value="under_review">Under Review</option>
          <option value="done">Done</option>
        </select>

        <select
          value={filterPriority}
          onChange={(e) => {
            setFilterPriority(e.target.value);
            setPage(1);
          }}
          className="px-3 py-2 border border-slate-200 rounded-lg text-sm text-slate-600 focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          <option value="">All Priorities</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>

        <select
          value={sortBy}
          onChange={(e) => {
            setSortBy(e.target.value);
            setPage(1);
          }}
          className="px-3 py-2 border border-slate-200 rounded-lg text-sm text-slate-600 focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          <option value="newest">Newest First</option>
          <option value="oldest">Oldest First</option>
        </select>

        {isFetching && (
          <div className="absolute -bottom-6 right-2 text-xs font-medium text-brand-600 animate-pulse">
            Updating...
          </div>
        )}
      </div>

      {/* Kanban Columns */}
      <div className="flex-1 min-h-96 flex gap-6 overflow-x-auto overflow-y-hidden pb-4">
        {COLUMNS.map((col) => {
          const columnTasks = tasks.filter((t) => t.status === col.id);

          return (
            <div
              key={col.id}
              className="w-80 shrink-0 min-h-0 flex flex-col bg-slate-100 rounded-xl p-4"
            >
              <div className="flex justify-between items-center mb-4">
                <h3 className="font-semibold text-slate-700">
                  {col.title}{" "}
                  <span className="text-slate-400 text-sm ml-1">
                    {columnTasks.length}
                  </span>
                </h3>
                <MoreHorizontal className="w-5 h-5 text-slate-400" />
              </div>

              <div className="flex-1 min-h-0 space-y-3 overflow-y-auto">
                {columnTasks.map((task) => {
                  const isOverdue =
                    task.dueDate &&
                    new Date(task.dueDate) < new Date() &&
                    task.status !== "done";

                  return (
                    <div
                      key={task.id}
                      onClick={() => setSelectedTask(task)}
                      className="bg-white p-4 rounded-lg shadow-sm border border-slate-200 hover:border-brand-400 transition-colors group cursor-pointer"
                    >
                      <div className="flex items-start justify-between gap-2 mb-2">
                        <h4 className="font-medium text-slate-900">
                          {task.title}
                        </h4>
                        <button
                          title="Delete task"
                          onClick={(e) => {
                            e.stopPropagation();
                            if (
                              window.confirm(`Delete task "${task.title}"?`)
                            ) {
                              deleteTaskMutation.mutate(task.id);
                            }
                          }}
                          className="p-1 text-slate-300 hover:text-red-600 hover:bg-red-50 rounded transition-colors shrink-0"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                      <p className="text-xs text-slate-500 line-clamp-2 mb-3">
                        {task.description}
                      </p>

                      {task.dueDate && (
                        <div
                          className={`text-xs mb-3 flex items-center gap-1 font-medium ${isOverdue ? "text-red-600" : "text-slate-500"}`}
                        >
                          <Clock className="w-3 h-3" />
                          {isOverdue ? "Overdue: " : "Due: "}
                          {new Date(task.dueDate).toLocaleDateString()}
                        </div>
                      )}

                      {task.subtasks && task.subtasks.length > 0 && (
                        <div className="mb-3">
                          <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
                            <span>Subtasks</span>
                            <span>
                              {task.subtasks.filter((s) => s.completed).length}/
                              {task.subtasks.length}
                            </span>
                          </div>
                          <div className="w-full bg-slate-100 rounded-full h-1.5">
                            <div
                              className="bg-brand-500 h-1.5 rounded-full"
                              style={{
                                width: `${(task.subtasks.filter((s) => s.completed).length / task.subtasks.length) * 100}%`,
                              }}
                            ></div>
                          </div>
                        </div>
                      )}

                      <div className="flex items-center justify-between mt-4">
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

                        <div className="flex items-center gap-2">
                          {task.assigneeId && (
                            <div className="flex items-center gap-1.5">
                              <div
                                className="w-6 h-6 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center text-xs font-bold"
                                title={
                                  members?.find((m) => m.id === task.assigneeId)
                                    ?.name || "Assigned"
                                }
                              >
                                {members
                                  ?.find((m) => m.id === task.assigneeId)
                                  ?.name.charAt(0) || "U"}
                              </div>
                            </div>
                          )}

                          {user?.role !== "viewer" ? (
                            <select
                              onClick={(e) => e.stopPropagation()}
                              value={task.status}
                              onChange={(e) =>
                                updateStatusMutation.mutate({
                                  taskId: task.id,
                                  status: e.target.value,
                                })
                              }
                              className="text-xs bg-slate-50 border border-slate-200 rounded px-1 py-1 text-slate-600 outline-none cursor-pointer hover:bg-slate-100"
                            >
                              <option value="todo">To Do</option>
                              <option value="in_progress">In Progress</option>
                              <option value="under_review">Under Review</option>
                              <option value="done">Done</option>
                            </select>
                          ) : (
                            <span className="text-xs font-medium text-slate-400 capitalize">
                              {task.status.replace("_", " ")}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
                {columnTasks.length === 0 && (
                  <div className="text-center p-4 text-sm text-slate-400 border-2 border-dashed border-slate-200 rounded-lg">
                    No tasks found
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between border-t border-slate-200 pt-4 mt-2">
        <span className="text-sm text-slate-500">
          Showing{" "}
          <span className="font-medium text-slate-900">
            {totalCount === 0 ? 0 : (page - 1) * limit + 1}
          </span>{" "}
          to{" "}
          <span className="font-medium text-slate-900">
            {Math.min(page * limit, totalCount)}
          </span>{" "}
          of <span className="font-medium text-slate-900">{totalCount}</span>{" "}
          tasks
        </span>
        <div className="flex items-center gap-2">
          <button
            disabled={page === 1}
            onClick={() => setPage((p) => p - 1)}
            className="p-2 border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-50 text-slate-600"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <span className="text-sm font-medium text-slate-700 px-2">
            Page {page} of {totalPages || 1}
          </span>
          <button
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="p-2 border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-50 text-slate-600"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Create Task Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-bold text-slate-900 mb-4">
              Create New Task
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Task Title <span className="text-red-500">*</span>
                </label>
                <input
                  required
                  type="text"
                  value={formData.title}
                  onChange={(e) =>
                    setFormData({ ...formData, title: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500"
                  placeholder="e.g. Implement login API"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Description
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500"
                  rows={2}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Priority
                </label>
                <select
                  value={formData.priority}
                  onChange={(e) =>
                    setFormData({ ...formData, priority: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 bg-white"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Assignee
                </label>
                <select
                  value={formData.assigneeId}
                  onChange={(e) =>
                    setFormData({ ...formData, assigneeId: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 bg-white"
                >
                  <option value="">Unassigned</option>
                  {members?.map((member) => (
                    <option key={member.id} value={member.id}>
                      {member.name} ({member.role})
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Due Date
                </label>
                <input
                  type="date"
                  value={formData.dueDate}
                  onChange={(e) =>
                    setFormData({ ...formData, dueDate: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500"
                />
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50"
                >
                  {createMutation.isPending ? "Creating..." : "Create Task"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* AI Magic Modal */}
      {isAIModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-bold text-slate-900 mb-2 flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-purple-600" /> AI Task Creator
            </h2>
            <p className="text-sm text-slate-500 mb-4">
              Describe what needs to be done. AI will create the task with its
              description, priority, status, assignee, due date and subtasks,
              just like filling the form by hand.
            </p>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                aiMutation.mutate(aiPrompt);
              }}
            >
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Your prompt <span className="text-red-500">*</span>
              </label>
              <textarea
                required
                value={aiPrompt}
                onChange={(e) => setAiPrompt(e.target.value)}
                placeholder='e.g. "Urgent: fix the login bug, assign it to Richard, due next Friday. Needs reproducing, fixing and testing."'
                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-purple-500 mb-4"
                rows={4}
              />
              <p className="text-xs text-slate-400 -mt-2 mb-4">
                Tip: mention a name to assign it, a date like
                &quot;tomorrow&quot; or &quot;next Friday&quot;, a priority, or
                list your own steps for subtasks.
              </p>

              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setIsAIModalOpen(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={aiMutation.isPending || !aiPrompt.trim()}
                  className="px-4 py-2 bg-purple-600 text-white text-sm font-medium rounded-lg hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"
                >
                  {aiMutation.isPending ? "Analyzing prompt..." : "Create Task"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Task Details Drawer */}
      {selectedTask && (
        <TaskDrawer
          task={selectedTask}
          projectId={projectId}
          members={members}
          onClose={() => setSelectedTask(null)}
        />
      )}
    </div>
  );
}
