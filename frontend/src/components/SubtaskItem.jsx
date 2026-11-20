import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { toast } from "sonner";
import { ChevronRight, ChevronDown, Send, MessageSquare } from "lucide-react";

export default function SubtaskItem({
  taskId,
  subtask,
  members,
  onToggle,
  onUpdated,
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState(subtask.title);
  const [description, setDescription] = useState(subtask.description || "");
  const [comment, setComment] = useState("");

  // Temp subtasks (optimistic) have no real ID on the server yet
  const isTemp = String(subtask.id).startsWith("temp-");

  // Keep the editor in sync if the subtask changes from outside
  useEffect(() => {
    setTitle(subtask.title);
    setDescription(subtask.description || "");
  }, [subtask.title, subtask.description]);

  const isDirty =
    title.trim() !== subtask.title ||
    description !== (subtask.description || "");

  // ---- Comments for this subtask ----
  const { data: comments, isLoading } = useQuery({
    queryKey: ["subtask-comments", taskId, subtask.id],
    queryFn: async () => {
      const res = await api.get(
        `/tasks/${taskId}/subtasks/${subtask.id}/comments`,
      );
      return res.data;
    },
    enabled: open && !isTemp,
  });

  const addCommentMutation = useMutation({
    mutationFn: (content) =>
      api.post(`/tasks/${taskId}/subtasks/${subtask.id}/comments`, {
        content,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["subtask-comments", taskId, subtask.id],
      });
      queryClient.invalidateQueries({ queryKey: ["activities", taskId] });
      setComment("");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to add comment"),
  });

  // ---- Save title + description ----
  const saveMutation = useMutation({
    mutationFn: ({ title, description }) =>
      api.patch(`/tasks/${taskId}/subtasks/${subtask.id}/details`, {
        title,
        description,
      }),
    onSuccess: (res) => {
      onUpdated({ ...subtask, ...res.data });
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      toast.success("Subtask updated!");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to update subtask"),
  });

  const handleSave = () => {
    if (!title.trim()) {
      toast.error("Title cannot be empty");
      return;
    }
    saveMutation.mutate({ title: title.trim(), description });
  };

  const handleComment = (e) => {
    e.preventDefault();
    if (!comment.trim()) return;
    addCommentMutation.mutate(comment.trim());
  };

  return (
    <div className="border border-slate-200 rounded-lg overflow-hidden">
      {/* Row */}
      <div className="flex items-center gap-2 p-2 hover:bg-slate-50 transition-colors">
        <input
          type="checkbox"
          checked={subtask.completed}
          onChange={(e) => onToggle(subtask.id, e.target.checked)}
          disabled={isTemp}
          className="w-4 h-4 text-brand-600 rounded border-slate-300 focus:ring-brand-500"
        />
        <button
          type="button"
          disabled={isTemp}
          onClick={() => setOpen((o) => !o)}
          className="flex-1 flex items-center gap-2 text-left disabled:cursor-wait"
        >
          <span
            className={`text-sm flex-1 ${subtask.completed ? "line-through text-slate-400" : "text-slate-700"}`}
          >
            {subtask.title}
          </span>
          {subtask.description && (
            <span className="text-[10px] text-slate-400">has notes</span>
          )}
          {open ? (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronRight className="w-4 h-4 text-slate-400" />
          )}
        </button>
      </div>

      {/* Details panel */}
      {open && (
        <div className="border-t border-slate-200 bg-slate-50 p-3 space-y-4">
          {/* Title + Description */}
          <div className="space-y-2">
            <label className="block text-xs font-semibold text-slate-600">
              Title <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-1.5 text-sm border border-slate-300 rounded-lg bg-white focus:ring-2 focus:ring-brand-500"
            />
            <label className="block text-xs font-semibold text-slate-600">
              Description
            </label>
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Add more details about this subtask..."
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white focus:ring-2 focus:ring-brand-500 resize-none"
            />
            <div className="flex justify-end">
              <button
                type="button"
                onClick={handleSave}
                disabled={!isDirty || saveMutation.isPending}
                className="px-3 py-1.5 bg-brand-600 text-white text-xs font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                {saveMutation.isPending ? "Saving..." : "Save changes"}
              </button>
            </div>
          </div>

          {/* Comments */}
          <div>
            <h4 className="text-xs font-semibold text-slate-600 mb-2 flex items-center gap-1.5">
              <MessageSquare className="w-3.5 h-3.5" />
              Comments
              <span className="bg-white border border-slate-200 text-slate-600 px-1.5 py-0.5 rounded-full text-[10px]">
                {comments?.length || 0}
              </span>
            </h4>

            <div className="space-y-3 mb-3">
              {isLoading ? (
                <p className="text-xs text-slate-400">Loading comments...</p>
              ) : !comments || comments.length === 0 ? (
                <p className="text-xs text-slate-400">No comments yet.</p>
              ) : (
                comments.map((c) => {
                  const author = members?.find((m) => m.id === c.userId);
                  return (
                    <div key={c.id} className="flex gap-2">
                      <div className="w-6 h-6 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-bold text-[10px] shrink-0">
                        {author?.name?.charAt(0) || "U"}
                      </div>
                      <div className="flex-1 bg-white border border-slate-200 p-2 rounded-lg rounded-tl-none">
                        <div className="flex justify-between items-center mb-0.5">
                          <span className="text-xs font-semibold text-slate-700">
                            {author?.name || "User"}
                          </span>
                          <span className="text-[10px] text-slate-400">
                            {new Date(c.createdAt).toLocaleString([], {
                              dateStyle: "short",
                              timeStyle: "short",
                            })}
                          </span>
                        </div>
                        <p className="text-xs text-slate-700 whitespace-pre-wrap">
                          {c.content}
                        </p>
                      </div>
                    </div>
                  );
                })
              )}
            </div>

            <form onSubmit={handleComment} className="flex gap-2">
              <div className="relative flex-1">
                <input
                  type="text"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Comment on this subtask..."
                  className="w-full pr-7 px-3 py-1.5 text-sm border border-slate-300 rounded-lg bg-white focus:ring-2 focus:ring-brand-500"
                />
                {!comment.trim() && (
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-red-500 text-sm pointer-events-none">
                    *
                  </span>
                )}
              </div>
              <button
                type="submit"
                disabled={addCommentMutation.isPending || !comment.trim()}
                className="bg-brand-600 text-white p-2 rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                <Send className="w-4 h-4" />
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
