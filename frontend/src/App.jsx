import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "sonner";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Dashboard from "./pages/Dashboard";
import Projects from "./pages/Projects";
import Layout from "./components/Layout";
import ProtectedRoute from "./components/ProtectedRoute";
import ProjectBoard from "./pages/ProjectBoard";
import Team from "./pages/Team";
import AcceptInvite from "./pages/AcceptInvite";
import MyTasks from "./pages/MyTasks";
import Activity from "./pages/Activity";

function App() {
  return (
    <BrowserRouter>
      <Toaster position="top-right" richColors />
      <Routes>
        {/* Public Routes */}
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/accept-invite" element={<AcceptInvite />} />

        {/* Wrap protected routes in the Layout */}
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/projects" element={<Projects />} />
            <Route path="/projects/:projectId" element={<ProjectBoard />} />
            <Route path="/team" element={<Team />} />
            <Route path="/tasks" element={<MyTasks />} />
            <Route path="/activity" element={<Activity />} />
          </Route>
        </Route>

        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
