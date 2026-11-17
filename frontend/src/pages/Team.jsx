import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { toast } from "sonner";
import { UserPlus } from "lucide-react";
import { useSelector } from "react-redux";

export default function Team() {
  const queryClient = useQueryClient();
  const { user } = useSelector((state) => state.auth); // To know who is currently logged in

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [inviteData, setInviteData] = useState({ email: "", role: "member" });

  // Fetch Team Members
  const { data: members, isLoading } = useQuery({
    queryKey: ["team"],
    queryFn: async () => {
      const res = await api.get("/team");
      return res.data;
    },
  });

  // Invite Mutation
  const inviteMutation = useMutation({
    mutationFn: (data) => api.post("/invites", data),
    onSuccess: () => {
      toast.success("Invitation email sent successfully!");
      setIsModalOpen(false);
      setInviteData({ email: "", role: "member" });
    },
    onError: (err) => {
      toast.error(err.response?.data?.error || "Failed to send invite");
    },
  });

  // Update Role Mutation
  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }) =>
      api.patch(`/team/${userId}/role`, { role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["team"] });
      toast.success("Role updated successfully!");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to update role"),
  });

  // Remove User Mutation
  const removeUserMutation = useMutation({
    mutationFn: (userId) => api.delete(`/team/${userId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["team"] });
      toast.success("User removed from team.");
    },
    onError: (err) =>
      toast.error(err.response?.data?.error || "Failed to remove user"),
  });

  const handleInvite = (e) => {
    e.preventDefault();
    inviteMutation.mutate(inviteData);
  };

  if (isLoading)
    return <div className="p-8 text-slate-500">Loading team...</div>;

  return (
    <div className="max-w-5xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Team Members</h1>
          <p className="text-slate-500 mt-1">
            Manage who has access to your workspace.
          </p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 bg-brand-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-brand-700 transition-colors"
        >
          <UserPlus className="w-4 h-4" /> Invite Member
        </button>
      </div>

      {/* Team Table */}
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-slate-50 border-b border-slate-200 text-sm text-slate-600">
              <th className="p-4 font-medium">Name</th>
              <th className="p-4 font-medium">Email</th>
              <th className="p-4 font-medium">Role</th>
              <th className="p-4 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {members?.map((member) => (
              <tr
                key={member.id}
                className="border-b border-slate-100 last:border-0 hover:bg-slate-50 transition-colors"
              >
                <td className="p-4 flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-bold text-sm">
                    {member.name.charAt(0)}
                  </div>
                  <span className="font-medium text-slate-900">
                    {member.name}
                  </span>
                </td>
                <td className="p-4 text-slate-600">{member.email}</td>
                <td className="p-4">
                  {/* Role Dropdown */}
                  <select
                    value={member.role}
                    onChange={(e) =>
                      updateRoleMutation.mutate({
                        userId: member.id,
                        role: e.target.value,
                      })
                    }
                    // Disable dropdown if the user is the Owner OR if it's the logged-in user's own row (to prevent self-demotion)
                    disabled={member.role === "owner" || member.id === user.id}
                    className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-slate-100 text-slate-800 capitalize border border-transparent hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <option value="owner">Owner</option>
                    <option value="admin">Admin</option>
                    <option value="member">Member</option>
                  </select>
                </td>
                <td className="p-4 text-right">
                  {/* Only show remove button if they are not the owner and not yourself */}
                  {member.role !== "owner" && member.id !== user.id && (
                    <button
                      onClick={() => {
                        if (
                          window.confirm(
                            `Are you sure you want to remove ${member.name}?`,
                          )
                        ) {
                          removeUserMutation.mutate(member.id);
                        }
                      }}
                      className="text-xs font-medium text-red-600 hover:text-red-700 px-3 py-1 bg-red-50 hover:bg-red-100 rounded-md transition-colors"
                    >
                      Remove
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Invite Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <h2 className="text-lg font-bold text-slate-900 mb-4">
              Invite Team Member
            </h2>

            <form onSubmit={handleInvite} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Email Address
                </label>
                <input
                  required
                  type="email"
                  value={inviteData.email}
                  onChange={(e) =>
                    setInviteData({ ...inviteData, email: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500"
                  placeholder="employee@company.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Role
                </label>
                <select
                  value={inviteData.role}
                  onChange={(e) =>
                    setInviteData({ ...inviteData, role: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 bg-white"
                >
                  <option value="member">Member (Can work on tasks)</option>
                  <option value="admin">
                    Admin (Can manage team & projects)
                  </option>
                </select>
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
                  disabled={inviteMutation.isPending}
                  className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50"
                >
                  {inviteMutation.isPending ? "Sending..." : "Send Invite"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
