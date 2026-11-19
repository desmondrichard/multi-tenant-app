import { useState, useEffect } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useSelector, useDispatch } from 'react-redux';
import { logout, updateToken } from '../store/authSlice';
import { LayoutDashboard, FolderKanban, CheckSquare, Users, LogOut, Bell, Check, Layers, ChevronDown } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export default function Layout() {
  const { user, token } = useSelector((state) => state.auth);
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  
  const [isNotifOpen, setIsNotifOpen] = useState(false);
  const [isWorkspaceOpen, setIsWorkspaceOpen] = useState(false);

  // Decode JWT to get the active workspace details
  let activeOrgId = "";
  let activeOrgName = "Workspace";
  if (token) {
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      activeOrgId = payload.organizationId;
      activeOrgName = payload.orgName || "Workspace";
    } catch (e) {
      console.error("Failed to decode token");
    }
  }

  // Fetch Notifications
  const { data: notifications } = useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const res = await api.get('/notifications');
      return res.data;
    },
    refetchInterval: 30000,
  });

  const unreadCount = notifications?.filter(n => !n.isRead).length || 0;

  const markReadMutation = useMutation({
    mutationFn: (id) => api.patch(`/notifications/${id}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const handleLogout = () => {
    dispatch(logout());
    navigate('/login');
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
    { name: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
    { name: 'Projects', path: '/projects', icon: FolderKanban },
    { name: 'My Tasks', path: '/tasks', icon: CheckSquare },
    { name: 'Team', path: '/team', icon: Users },
  ];

  return (
    <div className="min-h-screen flex bg-slate-50">
      {/* Sidebar */}
      <aside className="w-64 bg-white border-r border-slate-200 flex flex-col">
        <div className="h-16 flex items-center gap-2 px-6 border-b border-slate-200">
          <div className="w-8 h-8 bg-brand-600 rounded-lg flex items-center justify-center shadow-sm">
            <Layers className="w-5 h-5 text-white" />
          </div>
          <span className="text-xl font-bold text-slate-900">TaskFlow</span>
        </div>
        
        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive 
                    ? 'bg-brand-50 text-brand-700' 
                    : 'text-slate-600 hover:bg-slate-100'
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
      <div className="flex-1 flex flex-col">
        {/* Topbar */}
        <header className="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8">
          
          {/* Workspace Switcher */}
          <div className="relative">
            <button 
              onClick={() => setIsWorkspaceOpen(!isWorkspaceOpen)}
              className="flex items-center gap-2 text-sm text-slate-500 hover:text-slate-900 transition-colors"
            >
              Workspace: <span className="font-semibold text-slate-900">{activeOrgName}</span> <ChevronDown className="w-4 h-4" />
            </button>

            {isWorkspaceOpen && (
              <div className="absolute left-0 mt-2 w-64 bg-white rounded-xl shadow-lg border border-slate-200 z-50">
                <div className="p-2">
                  <p className="text-xs font-semibold text-slate-400 px-2 py-1 uppercase tracking-wider">Your Workspaces</p>
                  {user?.memberships?.map((m) => (
                    <button
                      key={m.organizationId}
                      onClick={() => handleSwitchWorkspace(m.organizationId)}
                      className={`w-full text-left px-2 py-2 rounded-lg text-sm flex justify-between items-center ${
                        activeOrgId === m.organizationId 
                          ? 'bg-brand-50 text-brand-700 font-medium' 
                          : 'hover:bg-slate-50 text-slate-700'
                      }`}
                    >
                      {m.organizationName}
                      {activeOrgId === m.organizationId && <Check className="w-4 h-4" />}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
          
          <div className="flex items-center gap-5">
            
            {/* Notification Bell */}
            <div className="relative">
              <button 
                onClick={() => setIsNotifOpen(!isNotifOpen)}
                className="p-2 text-slate-500 hover:bg-slate-100 rounded-full transition-colors relative"
              >
                <Bell className="w-5 h-5" />
                {unreadCount > 0 && (
                  <span className="absolute top-1.5 right-1.5 w-2.5 h-2.5 bg-red-500 rounded-full border-2 border-white"></span>
                )}
              </button>

              {/* Notification Dropdown */}
              {isNotifOpen && (
                <div className="absolute right-0 mt-2 w-80 bg-white rounded-xl shadow-lg border border-slate-200 z-50 animate-in fade-in slide-in-from-top-2">
                  <div className="p-3 border-b border-slate-100 flex justify-between items-center bg-slate-50 rounded-t-xl">
                    <h3 className="font-semibold text-slate-900 text-sm">Notifications</h3>
                    <span className="text-xs text-slate-500">{unreadCount} unread</span>
                  </div>
                  <div className="max-h-80 overflow-y-auto">
                    {notifications?.length === 0 ? (
                      <p className="p-4 text-sm text-slate-500 text-center">You're all caught up!</p>
                    ) : (
                      notifications?.map(notif => (
                        <div key={notif.id} className={`p-4 border-b border-slate-50 flex gap-3 transition-colors ${notif.isRead ? 'opacity-60 bg-white' : 'bg-brand-50/30'}`}>
                          <div className="flex-1">
                            <p className="text-sm font-medium text-slate-900">{notif.title}</p>
                            <p className="text-xs text-slate-500 mt-1">{notif.message}</p>
                          </div>
                          {!notif.isRead && (
                            <button 
                              onClick={() => markReadMutation.mutate(notif.id)}
                              className="text-brand-600 hover:text-brand-800 self-start mt-1"
                              title="Mark as read"
                            >
                              <Check className="w-4 h-4" />
                            </button>
                          )}
                        </div>
                      ))
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* User Profile */}
            <div className="flex items-center gap-3 border-l border-slate-200 pl-5">
              <div className="w-8 h-8 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-bold text-sm">
                {user?.name?.charAt(0)}
              </div>
              <span className="text-sm font-medium text-slate-700">{user?.name}</span>
            </div>
            
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 p-8 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}