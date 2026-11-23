import { useState, useEffect, useRef } from "react";
import { Outlet, NavLink, useNavigate, useLocation } from "react-router-dom";
import { useSelector, useDispatch } from "react-redux";
import { logout, updateToken } from "../store/authSlice";
import {
  LayoutDashboard,
  FolderKanban,
  CheckSquare,
  Users,
  LogOut,
  Bell,
  Check,
  CheckCheck,
  Layers,
  ChevronDown,
  Activity,
  Menu,
  X,
} from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { timeAgo, useMinuteTick } from "../lib/activity";
import AIChatbot from "./AIChatbot";

export default function Layout() {
  const { user } = useSelector((state) => state.auth);
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [isNotifOpen, setIsNotifOpen] = useState(false);
  const [isWorkspaceOpen, setIsWorkspaceOpen] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false); // mobile menu
  const [notifLimit, setNotifLimit] = useState(10); // notifications loaded so far
  const location = useLocation();

  // Close the mobile menu whenever the page changes
  useEffect(() => {
    setIsSidebarOpen(false);
  }, [location.pathname]);

  // Active workspace comes from the server profile (GET /me)
  const activeOrgId = user?.organizationId || "";
  const activeOrgName = user?.orgName || "Workspace";

  useMinuteTick(); // keeps "5 minutes ago" fresh
  const notifRef = useRef(null);

  // Fetch Notifications (10 at a time; "Load more" asks for 10 more).
  // Real-time: the WebSocket handler refreshes this on every NOTIFICATION event.
  // The 30s interval + refetch on focus are only a safety net.
  const { data: notifData, isFetching: isFetchingNotifs } = useQuery({
    queryKey: ["notifications", notifLimit],
    queryFn: async () => {
      const res = await api.get(`/notifications?limit=${notifLimit}`);
      return res.data;
    },
    placeholderData: (previous) => previous,
    refetchInterval: 30000,
    refetchOnWindowFocus: true,
  });

  const notifications = notifData?.notifications || [];
  const unreadCount = notifData?.unreadCount || 0;
  const hasMoreNotifs = notifData?.hasMore || false;

  // Close the dropdown when clicking outside of it
  useEffect(() => {
    const onClickOutside = (e) => {
      if (notifRef.current && !notifRef.current.contains(e.target)) {
        setIsNotifOpen(false);
      }
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  // Clicking the bell ALWAYS fetches the latest notifications
  const toggleNotifications = () => {
    const opening = !isNotifOpen;
    setIsNotifOpen(opening);
    if (opening) {
      setNotifLimit(10);
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  };

  const markReadMutation = useMutation({
    mutationFn: (id) => api.patch(`/notifications/${id}/read`),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: ["notifications"] });
      queryClient.setQueriesData(
        { queryKey: ["notifications"] },
        (old) =>
          old && {
            ...old,
            unreadCount: Math.max(0, (old.unreadCount || 0) - 1),
            notifications: old.notifications.map((n) =>
              n.id === id ? { ...n, isRead: true } : n,
            ),
          },
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: () => api.patch("/notifications/read-all"),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["notifications"] });
      queryClient.setQueriesData(
        { queryKey: ["notifications"] },
        (old) =>
          old && {
            ...old,
            unreadCount: 0,
            notifications: old.notifications.map((n) => ({
              ...n,
              isRead: true,
            })),
          },
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  // Click a notification: mark it read and open the task it is about
  const handleNotificationClick = (notif) => {
    if (!notif.isRead) markReadMutation.mutate(notif.id);
    setIsNotifOpen(false);
    if (notif.projectId && notif.taskId) {
      navigate(`/projects/${notif.projectId}?task=${notif.taskId}`);
    }
  };

  const handleLogout = () => {
    dispatch(logout());
    queryClient.clear(); // don't keep the previous user's cached data
    navigate("/login");
  };

  const handleSwitchWorkspace = async (orgId) => {
    try {
      const res = await api.post(`/workspaces/switch/${orgId}`);
      dispatch(updateToken({ token: res.data.token }));
      setIsWorkspaceOpen(false);
      // Force a reload to clear all TanStack queries and fetch data for the new workspace
      window.location.href = "/dashboard";
    } catch (error) {
      alert("Failed to switch workspace");
    }
  };

  const navItems = [
    { name: "Dashboard", path: "/dashboard", icon: LayoutDashboard },
    { name: "Projects", path: "/projects", icon: FolderKanban },
    { name: "My Tasks", path: "/tasks", icon: CheckSquare },
    { name: "Activity", path: "/activity", icon: Activity },
    { name: "Team", path: "/team", icon: Users },
  ];

  return (
    <div className="h-screen overflow-hidden flex bg-slate-50">
      {/* Dark overlay behind the mobile menu */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black/40 z-30 md:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar: slides in on mobile, always visible from md and up */}
      <aside
        className={`fixed inset-y-0 left-0 z-40 w-64 bg-white border-r border-slate-200 flex flex-col transform transition-transform duration-200 md:static md:translate-x-0 md:shrink-0 md:h-full ${
          isSidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="h-16 flex items-center justify-between gap-2 px-6 border-b border-slate-200">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-brand-600 rounded-lg flex items-center justify-center shadow-sm">
              <Layers className="w-5 h-5 text-white" />
            </div>
            <span className="text-xl font-bold text-slate-900">TaskFlow</span>
          </div>
          <button
            onClick={() => setIsSidebarOpen(false)}
            className="md:hidden p-1 text-slate-500 hover:bg-slate-100 rounded"
            aria-label="Close menu"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className="flex-1 min-h-0 p-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-brand-50 text-brand-700"
                    : "text-slate-600 hover:bg-slate-100"
                }`
              }
            >
              <item.icon className="w-4 h-4" />
              {item.name}
            </NavLink>
          ))}
        </nav>

        <div className="p-4 border-t border-slate-200">
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-3 py-2 w-full rounded-lg text-sm font-medium text-slate-600 hover:bg-red-50 hover:text-red-600 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Log out
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex-1 min-w-0 h-full flex flex-col">
        {/* Topbar */}
        <header className="h-16 shrink-0 bg-white border-b border-slate-200 flex items-center justify-between gap-2 px-4 md:px-8">
          {/* Hamburger (mobile only) */}
          <button
            onClick={() => setIsSidebarOpen(true)}
            className="md:hidden p-2 -ml-2 text-slate-600 hover:bg-slate-100 rounded-lg"
            aria-label="Open menu"
          >
            <Menu className="w-5 h-5" />
          </button>

          {/* Workspace Switcher */}
          <div className="relative min-w-0 flex-1 md:flex-none">
            <button
              onClick={() => setIsWorkspaceOpen(!isWorkspaceOpen)}
              className="flex items-center gap-2 max-w-full text-sm text-slate-500 hover:text-slate-900 transition-colors"
            >
              <span className="hidden sm:inline">Workspace:</span>
              <span className="font-semibold text-slate-900 truncate">
                {activeOrgName}
              </span>
              <ChevronDown className="w-4 h-4 shrink-0" />
            </button>

            {isWorkspaceOpen && (
              <div className="absolute left-0 mt-2 w-64 max-w-[calc(100vw-2rem)] bg-white rounded-xl shadow-lg border border-slate-200 z-50">
                <div className="p-2">
                  <p className="text-xs font-semibold text-slate-400 px-2 py-1 uppercase tracking-wider">
                    Your Workspaces
                  </p>
                  {user?.memberships?.map((m) => (
                    <button
                      key={m.organizationId}
                      onClick={() => handleSwitchWorkspace(m.organizationId)}
                      className={`w-full text-left px-2 py-2 rounded-lg text-sm flex justify-between items-center ${
                        activeOrgId === m.organizationId
                          ? "bg-brand-50 text-brand-700 font-medium"
                          : "hover:bg-slate-50 text-slate-700"
                      }`}
                    >
                      {m.organizationName}
                      {activeOrgId === m.organizationId && (
                        <Check className="w-4 h-4" />
                      )}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 sm:gap-5">
            {/* Notification Bell */}
            <div className="relative" ref={notifRef}>
              <button
                onClick={toggleNotifications}
                className="p-2 text-slate-500 hover:bg-slate-100 rounded-full transition-colors relative"
              >
                <Bell className="w-5 h-5" />
                {unreadCount > 0 && (
                  <span className="absolute -top-0.5 -right-0.5 min-w-18px h-4.5 px-1 bg-red-500 text-white text-[10px] font-bold rounded-full border-2 border-white flex items-center justify-center">
                    {unreadCount > 9 ? "9+" : unreadCount}
                  </span>
                )}
              </button>

              {/* Notification Dropdown */}
              {isNotifOpen && (
                <div className="fixed left-3 right-3 top-18 sm:absolute sm:left-auto sm:right-0 sm:top-auto sm:mt-2 sm:w-96 bg-white rounded-xl shadow-lg border border-slate-200 z-50">
                  <div className="p-3 border-b border-slate-100 flex justify-between items-center bg-slate-50 rounded-t-xl">
                    <div>
                      <h3 className="font-semibold text-slate-900 text-sm">
                        Notifications
                      </h3>
                      <span className="text-xs text-slate-500">
                        {isFetchingNotifs
                          ? "Updating..."
                          : `${unreadCount} unread`}
                      </span>
                    </div>
                    {unreadCount > 0 && (
                      <button
                        onClick={() => markAllReadMutation.mutate()}
                        className="flex items-center gap-1 text-xs font-medium text-brand-600 hover:text-brand-800"
                      >
                        <CheckCheck className="w-4 h-4" /> Mark all as read
                      </button>
                    )}
                  </div>
                  <div className="max-h-96 overflow-y-auto">
                    {!notifications || notifications.length === 0 ? (
                      <p className="p-4 text-sm text-slate-500 text-center">
                        You're all caught up!
                      </p>
                    ) : (
                      notifications.map((notif) => (
                        <button
                          key={notif.id}
                          onClick={() => handleNotificationClick(notif)}
                          className={`w-full text-left p-4 border-b border-slate-50 flex gap-3 transition-colors hover:bg-slate-50 ${notif.isRead ? "opacity-60 bg-white" : "bg-brand-50/30"}`}
                        >
                          <span
                            className={`mt-1.5 w-2 h-2 rounded-full shrink-0 ${notif.isRead ? "bg-transparent" : "bg-brand-500"}`}
                          />
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-slate-900">
                              {notif.title}
                            </p>
                            <p className="text-xs text-slate-500 mt-1">
                              {notif.message}
                            </p>
                            <p
                              className="text-[11px] text-slate-400 mt-1"
                              title={new Date(notif.createdAt).toLocaleString()}
                            >
                              {timeAgo(notif.createdAt)}
                            </p>
                          </div>
                        </button>
                      ))
                    )}

                    {hasMoreNotifs && (
                      <button
                        onClick={() => setNotifLimit((l) => l + 10)}
                        disabled={isFetchingNotifs}
                        className="w-full py-3 text-sm font-medium text-brand-600 hover:bg-slate-50 disabled:opacity-50"
                      >
                        {isFetchingNotifs ? "Loading..." : "Load more"}
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* User Profile */}
            <div className="flex items-center gap-3 sm:border-l sm:border-slate-200 sm:pl-5">
              <div className="w-8 h-8 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-bold text-sm">
                {user?.name?.charAt(0)}
              </div>
              <span className="hidden sm:inline text-sm font-medium text-slate-700">
                {user?.name}
              </span>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 min-h-0 min-w-0 p-4 md:p-8 overflow-y-auto overflow-x-hidden">
          <Outlet />
        </main>
      </div>

      {/* Workspace AI chat helper (bottom-right) */}
      <AIChatbot />
    </div>
  );
}
