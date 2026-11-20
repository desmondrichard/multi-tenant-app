import { useEffect } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useSelector, useDispatch } from "react-redux";
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import { setUser } from "../store/authSlice";

export default function ProtectedRoute() {
  const dispatch = useDispatch();
  const { isAuthenticated, token, user } = useSelector((state) => state.auth);

  // The browser only stores the token. The user's profile (name, email,
  // role, workspaces) is loaded from the server. This also checks that the
  // user still belongs to the workspace: if they were removed, the API
  // answers 401 and the interceptor logs them out.
  const { data, isError, error, refetch } = useQuery({
    queryKey: ["me", token],
    queryFn: async () => {
      const res = await api.get("/me");
      return res.data;
    },
    enabled: isAuthenticated,
    retry: false,
    refetchOnWindowFocus: true,
  });

  useEffect(() => {
    if (data) dispatch(setUser(data));
  }, [data, dispatch]);

  // If not logged in, redirect to login page
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  // Server/network problem (not a 401, that one logs out automatically)
  if (isError && error?.response?.status !== 401) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-3 text-slate-600">
        <p>Could not load your account. Please check your connection.</p>
        <button
          onClick={() => refetch()}
          className="px-4 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700"
        >
          Try again
        </button>
      </div>
    );
  }

  // Wait until the profile is loaded
  if (!user) {
    return (
      <div className="min-h-screen flex items-center justify-center text-slate-500">
        Loading...
      </div>
    );
  }

  // Logged in and verified: render the child routes
  return <Outlet />;
}