import { useState } from "react";
import { Link } from "react-router-dom"; // <-- Added this import
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { toast } from "sonner";
import { Plus, FolderKanban } from "lucide-react";

export default function Projects() {
  const queryClient = useQueryClient();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    key: "",
    description: "",
  });

  // 1. Fetch Projects
  const { data: projects, isLoading } = useQuery({
    queryKey: ["projects"],
    queryFn: async () => {
      const res = await api.get("/projects");
      return res.data;
    },
  });

  // 2. Create Project Mutation
  const createMutation = useMutation({
    mutationFn: (newProject) => api.post("/projects", newProject),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] }); // Refresh list
      toast.success("Project created!");
      setIsModalOpen(false);
      setFormData({ name: "", key: "", description: "" });
    },
    onError: (error) => {
      toast.error(error.response?.data?.error || "Failed to create project");
    },
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    createMutation.mutate(formData);
  };

  if (isLoading)
    return <div className="text-slate-500">Loading projects...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Projects</h1>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 bg-brand-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-brand-700 transition-colors"
        >
          <Plus className="w-4 h-4" /> New Project
        </button>
      </div>

      {/* Projects Grid */}
      {projects?.length === 0 ? (
        <div className="text-center py-20 bg-white rounded-xl border border-slate-200 border-dashed">
          <FolderKanban className="w-12 h-12 text-slate-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-slate-900">
            No projects yet
          </h3>
          <p className="text-slate-500 mb-4">
            Create your first project to get started.
          </p>
          <button
            onClick={() => setIsModalOpen(true)}
            className="text-brand-600 font-medium hover:underline"
          >
            + Create Project
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects?.map((project) => (
            // Wrapped in a Link to navigate to the Kanban Board
            <Link to={`/projects/${project.id}`} key={project.id}>
              <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:shadow-md hover:border-brand-500 transition-all cursor-pointer h-full">
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 rounded-lg bg-brand-50 text-brand-600 flex items-center justify-center font-bold">
                    {project.key}
                  </div>
                  <h3 className="font-semibold text-slate-900">
                    {project.name}
                  </h3>
                </div>
                <p className="text-sm text-slate-500 line-clamp-2">
                  {project.description || "No description provided."}
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create Project Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-bold text-slate-900 mb-4">
              Create New Project
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Project Name <span className="text-red-500">*</span>
                </label>
                <input
                  required
                  type="text"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                  placeholder="E-commerce App"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Key (e.g. ECOM) <span className="text-red-500">*</span>
                </label>
                <input
                  required
                  type="text"
                  maxLength={8}
                  value={formData.key}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      key: e.target.value.toUpperCase(),
                    })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500 uppercase"
                  placeholder="ECOM"
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
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
                  rows={3}
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
                  {createMutation.isPending ? "Creating..." : "Create Project"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
