import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { toast } from "sonner";
import { X, Send, User, Clock, Trash2 } from "lucide-react";
import SubtaskItem from "./SubtaskItem";

export default function TaskDrawer({ task, projectId, members, onClose }) {
  const queryClient = useQueryClient();
  const [content, setContent] = useState("");
  const [newSubtask, setNewSubtask] = useState("");
  const [activeTab, setActiveTab] = useState("details"); // details | comments | activity

  // Local state for subtasks to make the UI instant
  const [localSubtasks, setLocalSubtasks] = useState(task.subtasks || []);

  // Sync local state if the parent task updates
  useEffect(() => {
    setLocalSubtasks(task.subtasks || []);
  }, [task.subtasks]);

  const { data: comments, isLoading: isLoadingComments } = useQuery({
    queryKey: ["comments", task.id],
    queryFn: async () => {
      const res = await api.get(`/tasks/${task.id}/comments`);
      return res.data;
    },
  });

  const { data: activities, isLoading: isLoadingActivities } = useQuery({
    queryKey: ["activities", task.id],
    queryFn: async () => {
      const res = await api.get(`/tasks/${task.id}/activities`);
      return res.data;
    },
  });

  const createCommentMutation = useMutation({
    mutationFn: (newComment) =>
      api.post(`/tasks/${task.id}/comments`, newComment),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["comments", task.id] });
      queryClient.invalidateQueries({ queryKey: ["activities", task.id] });
      setContent("");
      toast.success("Comment added!");
    },
    onError: () => toast.error("Failed to add comment"),
  });

  // Add Subtask Mutation (Optimistic)
  const addSubtaskMutation = useMutation({
    mutationFn: (title) => api.post(`/tasks/${task.id}/subtasks`, { title }),
    onMutate: async (newTitle) => {
      await queryClient.cancelQueries({ queryKey: ["tasks"] });
      const newSubtaskObj = {
        id: `temp-${Date.now()}`,
        title: newTitle,
        description: "",
        completed: false,
      };
      setLocalSubtasks((prev) => [...prev, newSubtaskObj]);
      setNewSubtask("");
      return { newSubtaskObj };
    },
    onError: (err, newTitle, context) => {
      setLocalSubtasks((prev) =>
        prev.filter((s) => s.id !== context.newSubtaskObj.id),
      );
      toast.error("Failed to add subtask");
    },
    onSuccess: (res) => {
      setLocalSubtasks((prev) =>
        prev.map((s) => (s.id.startsWith("temp-") ? res.data : s)),
      );
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      toast.success("Subtask added!");
    },
  });

  // Toggle Subtask Mutation
  const toggleSubtaskMutation = useMutation({
    mutationFn: ({ subtaskId, completed }) =>
      api.patch(`/tasks/${task.id}/subtasks/${subtaskId}`, { completed }),
    onSuccess: (_, variables) => {
      setLocalSubtasks((prev) =>
        prev.map((s) =>
          s.id === variables.subtaskId
            ? { ...s, completed: variables.completed }
            : s,
        ),
      );
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
    onError: () => {
      toast.error("Failed to update subtask");
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

  // Soft delete: the task is hidden but stays in the database
  const deleteTaskMutation = useMutation({
    mutationFn: () => api.delete(`/projects/${projectId}/tasks/${task.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
      queryClient.invalidateQueries({ queryKey: ["my-tasks"] });
      toast.success("Task deleted");
      onClose();
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to delete task"),
  });

  const handleDelete = () => {
    if (window.confirm(`Delete task "${task.title}"?`)) {
      deleteTaskMutation.mutate();
    }
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!content.trim()) return;
    createCommentMutation.mutate({ content });
  };

  const assignee = members?.find((m) => m.id === task.assigneeId);
  const isOverdue =
    task.dueDate &&
    new Date(task.dueDate) < new Date() &&
    task.status !== "done";

  const tabs = [
    { id: "details", label: "Details" },
    { id: "comments", label: "Comments", count: comments?.length || 0 },
    { id: "activity", label: "Activity" },
  ];

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/30 backdrop-blur-sm transition-opacity">
      <div className="w-full max-w-lg bg-white h-full shadow-2xl flex flex-col animate-slide-in">
        {/* Header */}
        <div className="p-6 border-b border-slate-200 flex justify-between items-start">
          <div>
            <h2 className="text-xl font-bold text-slate-900">{task.title}</h2>
            <div className="flex items-center gap-4 mt-3 text-sm text-slate-500">
              <span className="flex items-center gap-1">
                <User className="w-4 h-4" />
                {assignee ? assignee.name : "Unassigned"}
              </span>
              <span
                className={`font-bold uppercase px-2 py-1 rounded-md text-[10px] ${
                  task.priority === "high"
                    ? "bg-red-100 text-red-700"
                    : task.priority === "medium"
                      ? "bg-yellow-100 text-yellow-700"
                      : "bg-slate-100 text-slate-600"
                }`}
              >
                {task.priority}
              </span>
              {task.dueDate && (
                <span
                  className={`flex items-center gap-1 font-medium ${isOverdue ? "text-red-600" : ""}`}
                >
                  <Clock className="w-4 h-4" />
                  {new Date(task.dueDate).toLocaleDateString()}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={handleDelete}
              disabled={deleteTaskMutation.isPending}
              title="Delete task"
              className="p-2 hover:bg-red-50 rounded-full transition-colors disabled:opacity-50"
            >
              <Trash2 className="w-5 h-5 text-red-500" />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-slate-100 rounded-full transition-colors"
            >
              <X className="w-5 h-5 text-slate-500" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-200 px-6">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-3 text-sm font-medium border-b-2 -mb-px transition-colors flex items-center gap-2 ${
                activeTab === tab.id
                  ? "border-brand-600 text-brand-600"
                  : "border-transparent text-slate-500 hover:text-slate-700"
              }`}
            >
              {tab.label}
              {tab.count > 0 && (
                <span className="bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full text-xs">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-8">
          {/* ---------- DETAILS TAB ---------- */}
          {activeTab === "details" && (
            <>
              {/* Description */}
              <div>
                <h3 className="text-sm font-semibold text-slate-900 mb-2">
                  Description
                </h3>
                <p className="text-sm text-slate-700 bg-slate-50 p-4 rounded-lg">
                  {task.description || "No description provided."}
                </p>
              </div>

              {/* Subtasks */}
              <div>
                <div className="flex justify-between items-center mb-2">
                  <h3 className="text-sm font-semibold text-slate-900">
                    Subtasks
                  </h3>
                  <span className="text-xs text-slate-500">
                    {localSubtasks.filter((s) => s.completed).length} /{" "}
                    {localSubtasks.length} done
                  </span>
                </div>

                <div className="space-y-2 mb-3">
                  {localSubtasks.map((subtask) => (
                    <SubtaskItem
                      key={subtask.id}
                      taskId={task.id}
                      subtask={subtask}
                      members={members}
                      onToggle={(subtaskId, completed) =>
                        toggleSubtaskMutation.mutate({ subtaskId, completed })
                      }
                      onUpdated={(updated) =>
                        setLocalSubtasks((prev) =>
                          prev.map((s) => (s.id === updated.id ? updated : s)),
                        )
                      }
                    />
                  ))}
                </div>

                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    if (newSubtask.trim())
                      addSubtaskMutation.mutate(newSubtask);
                  }}
                  className="flex gap-2"
                >
                  <div className="relative flex-1">
                    <input
                      type="text"
                      value={newSubtask}
                      onChange={(e) => setNewSubtask(e.target.value)}
                      placeholder="Add a subtask..."
                      className="w-full pr-7 px-3 py-1.5 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500"
                    />
                    {!newSubtask.trim() && (
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 text-red-500 text-sm pointer-events-none">
                        *
                      </span>
                    )}
                  </div>
                  <button
                    type="submit"
                    disabled={!newSubtask.trim()}
                    className="text-sm font-medium text-brand-600 hover:text-brand-700 px-2 disabled:opacity-50"
                  >
                    Add
                  </button>
                </form>
              </div>
            </>
          )}

          {/* ---------- COMMENTS TAB ---------- */}
          {activeTab === "comments" && (
            <div className="space-y-4">
              {isLoadingComments ? (
                <p className="text-sm text-slate-400">Loading comments...</p>
              ) : comments?.length === 0 ? (
                <p className="text-sm text-slate-400">
                  No comments yet. Start the conversation!
                </p>
              ) : (
                comments?.map((comment) => {
                  const commentUser = members?.find(
                    (m) => m.id === comment.userId,
                  );
                  return (
                    <div key={comment.id} className="flex gap-3">
                      <div className="w-8 h-8 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-bold text-xs shrink-0">
                        {commentUser?.name.charAt(0) || "U"}
                      </div>
                      <div className="flex-1 bg-slate-50 p-3 rounded-lg rounded-tl-none">
                        <div className="flex justify-between items-center mb-1">
                          <span className="text-xs font-semibold text-slate-700">
                            {commentUser?.name || "User"}
                          </span>
                          <span className="text-[10px] text-slate-400">
                            {new Date(comment.createdAt).toLocaleTimeString(
                              [],
                              { hour: "2-digit", minute: "2-digit" },
                            )}
                          </span>
                        </div>
                        <p className="text-sm text-slate-700">
                          {comment.content}
                        </p>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          )}

          {/* ---------- ACTIVITY TAB ---------- */}
          {activeTab === "activity" && (
            <div className="space-y-4">
              {isLoadingActivities ? (
                <p className="text-sm text-slate-400">Loading activity...</p>
              ) : activities?.length === 0 ? (
                <p className="text-sm text-slate-400">No activity yet.</p>
              ) : (
                activities?.map((activity) => {
                  const activityUser = members?.find(
                    (m) => m.id === activity.userId,
                  );
                  return (
                    <div key={activity.id} className="flex gap-3 text-sm">
                      <div className="w-6 h-6 rounded-full bg-slate-100 text-slate-600 flex items-center justify-center font-bold text-[10px] shrink-0 mt-0.5">
                        {activityUser?.name.charAt(0) || "U"}
                      </div>
                      <div>
                        <span className="font-medium text-slate-900">
                          {activityUser?.name || "User"}
                        </span>
                        <span className="text-slate-500">
                          {" "}
                          {activity.details}
                        </span>
                        <p className="text-[10px] text-slate-400">
                          {new Date(activity.createdAt).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          )}
        </div>

        {/* Comment input (only on the Comments tab) */}
        {activeTab === "comments" && (
          <div className="p-4 border-t border-slate-200 bg-white">
            <form onSubmit={handleSubmit} className="flex gap-2">
              <div className="relative flex-1">
                <input
                  type="text"
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Write a comment..."
                  className="w-full pr-8 px-4 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-brand-500 text-sm"
                />
                {!content.trim() && (
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-red-500 text-sm pointer-events-none">
                    *
                  </span>
                )}
              </div>
              <button
                type="submit"
                disabled={createCommentMutation.isPending || !content.trim()}
                className="bg-brand-600 text-white p-2 rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                <Send className="w-5 h-5" />
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
