import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import { useSelector } from "react-redux";
import {
  FolderKanban,
  CheckSquare,
  Clock,
  Zap,
  ArrowRight,
} from "lucide-react";
import ActivityIcon from "../components/ActivityIcon";
import {
  describeActivity,
  timeAgo,
  activityLink,
  useMinuteTick,
} from "../lib/activity";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";

export default function Dashboard() {
  const { user } = useSelector((state) => state.auth);

  const { data: stats, isLoading } = useQuery({
    queryKey: ["analytics"],
    queryFn: async () => {
      const res = await api.get("/analytics");
      return res.data;
    },
  });

  // Live data: the WebSocket handler refreshes ["activity-feed"] on every change
  useMinuteTick();
  const { data: feed, isLoading: isLoadingActivity } = useQuery({
    queryKey: ["activity-feed", "recent"],
    queryFn: async () => {
      const res = await api.get("/activities?limit=5");
      return res.data;
    },
    refetchInterval: 60000, // safety net if the WebSocket is down
  });
  const recentActivities = feed?.activities || [];

  if (isLoading)
    return <div className="p-8 text-slate-500">Loading dashboard...</div>;

  const totalTasks = stats?.totalTasks || 0;
  const completedTasks =
    stats?.tasksByStatus?.find((s) => s.name === "Done")?.value || 0;

  const getPercentage = (val) => {
    if (totalTasks === 0) return 0;
    return Math.round((val / totalTasks) * 100);
  };

  return (
    <div className="min-h-full -m-4 p-4 md:-m-8 md:p-8 bg-dashboard-gradient relative overflow-hidden">
      <div className="flex flex-wrap justify-between items-start gap-4 mb-8 relative z-10">
        <div>
          <h1 className="text-3xl font-bold text-slate-900 flex items-center gap-2">
            Good morning, {user?.name}! <span className="text-3xl">👋</span>
          </h1>
          <p className="text-slate-500 mt-2 text-lg">
            Here's what's happening in your workspace today.
          </p>
        </div>
        <button className="flex items-center gap-2 bg-white/60 backdrop-blur-sm border border-slate-200 px-4 py-2 rounded-xl text-sm font-medium text-slate-700 shadow-sm hover:bg-white transition-colors">
          <Clock className="w-4 h-4 text-slate-500" /> This Month
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8 relative z-10">
        <div className="card-gradient-blue p-6 rounded-2xl soft-shadow border border-blue-100/50 flex items-center gap-5">
          <div className="w-14 h-14 rounded-xl bg-white shadow-sm flex items-center justify-center text-blue-500">
            <FolderKanban className="w-7 h-7" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-600">Total Projects</p>
            <h3 className="text-3xl font-bold text-slate-900 mt-1">
              {stats?.totalProjects || 0}
            </h3>
          </div>
        </div>

        <div className="card-gradient-green p-6 rounded-2xl soft-shadow border border-green-100/50 flex items-center gap-5">
          <div className="w-14 h-14 rounded-xl bg-white shadow-sm flex items-center justify-center text-green-500">
            <CheckSquare className="w-7 h-7" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-600">Total Tasks</p>
            <h3 className="text-3xl font-bold text-slate-900 mt-1">
              {totalTasks}
            </h3>
          </div>
        </div>

        <div className="card-gradient-orange p-6 rounded-2xl soft-shadow border border-orange-100/50 flex items-center gap-5">
          <div className="w-14 h-14 rounded-xl bg-white shadow-sm flex items-center justify-center text-orange-500">
            <Clock className="w-7 h-7" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-600">Completed</p>
            <h3 className="text-3xl font-bold text-slate-900 mt-1">
              {completedTasks}
            </h3>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 relative z-10">
        <div className="lg:col-span-2 bg-white p-6 rounded-2xl soft-shadow border border-slate-100">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-lg font-bold text-slate-900 flex items-center gap-2">
              <span className="w-2 h-5 bg-brand-500 rounded-full inline-block"></span>{" "}
              Task Distribution
            </h3>
          </div>

          <div className="flex flex-col md:flex-row items-center justify-center gap-10">
            <div className="relative w-56 h-56 sm:w-64 sm:h-64">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={stats?.tasksByStatus || []}
                    cx="50%"
                    cy="50%"
                    innerRadius={80}
                    outerRadius={100}
                    paddingAngle={2}
                    dataKey="value"
                    stroke="none"
                  >
                    {stats?.tasksByStatus?.map((entry, index) => (
                      <Cell
                        key={`cell-${index}`}
                        fill={
                          entry.name === "To Do"
                            ? "#94a3b8"
                            : entry.name === "In Progress"
                              ? "#3b82f6"
                              : entry.name === "Under Review"
                                ? "#f59e0b" // <-- Added
                                : "#22c55e"
                        }
                      />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="text-3xl font-bold text-slate-900">
                  {totalTasks}
                </span>
                <span className="text-sm text-slate-500">Total Tasks</span>
              </div>
            </div>

            <div className="space-y-5 flex-1 max-w-50">
              {stats?.tasksByStatus?.map((status) => (
                <div
                  key={status.name}
                  className="flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <div
                      className="w-3 h-3 rounded-full"
                      style={{
                        backgroundColor:
                          status.name === "To Do"
                            ? "#94a3b8"
                            : status.name === "In Progress"
                              ? "#3b82f6"
                              : status.name === "Under Review"
                                ? "#f59e0b" // <-- Added
                                : "#22c55e",
                      }}
                    ></div>
                    <span className="text-slate-600 font-medium">
                      {status.name}
                    </span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="font-bold text-slate-900">
                      {status.value}
                    </span>
                    <span className="text-sm text-slate-400 w-10 text-right">
                      {getPercentage(status.value)}%
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-2xl soft-shadow border border-slate-100">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-lg font-bold text-slate-900 flex items-center gap-2">
              <Zap className="w-5 h-5 text-brand-500" /> Recent Activity
            </h3>
            <Link
              to="/activity"
              className="text-sm font-medium text-brand-600 hover:text-brand-700 flex items-center gap-1"
            >
              View All <ArrowRight className="w-4 h-4" />
            </Link>
          </div>

          {isLoadingActivity ? (
            <p className="text-sm text-slate-400">Loading activity...</p>
          ) : recentActivities.length === 0 ? (
            <p className="text-sm text-slate-400">
              No activity yet. Create a task to get started!
            </p>
          ) : (
            <div className="relative border-l border-slate-200 ml-3 space-y-8 pb-4">
              {recentActivities.map((activity) => {
                const { type, text } = describeActivity(activity);
                const link = activityLink(activity);
                const content = (
                  <div>
                    <p className="text-sm text-slate-700">{text}</p>
                    <p
                      className="text-xs text-slate-400 mt-1"
                      title={new Date(activity.createdAt).toLocaleString()}
                    >
                      {timeAgo(activity.createdAt)}
                    </p>
                  </div>
                );
                return (
                  <div key={activity.id} className="relative pl-6">
                    <ActivityIcon type={type} className="absolute -left-3 top-1" />
                    {link ? (
                      <Link to={link} className="block hover:opacity-80">
                        {content}
                      </Link>
                    ) : (
                      content
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}