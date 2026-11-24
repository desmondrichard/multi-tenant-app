import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useNavigate } from "react-router-dom";
import { useDispatch } from "react-redux";
import { setCredentials } from "../store/authSlice";
import { api } from "../lib/api";
import { toast } from "sonner";

const registerSchema = z.object({
  orgName: z.string().min(2, "Organization name must be at least 2 characters"),
  slug: z
    .string()
    .min(2, "Slug must be at least 2 characters")
    .regex(
      /^[a-z0-9-]+$/,
      "Only lowercase letters, numbers, and hyphens allowed",
    ),
  name: z.string().min(2, "Name must be at least 2 characters"),
  email: z.string().email("Invalid email address"),
  password: z.string().min(6, "Password must be at least 6 characters"),
});

export default function Register() {
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();
  const dispatch = useDispatch();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(registerSchema),
  });

  const onSubmit = async (data) => {
    setIsLoading(true);
    try {
      const response = await api.post("/auth/register", data);

      dispatch(setCredentials({ token: response.data.token }));

      toast.success("Organization created successfully");
      navigate("/dashboard");
    } catch (error) {
      // This will now correctly show "workspace slug is already taken"
      toast.error(error.response?.data?.error || "Registration failed");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 p-4">
      <div className="w-full max-w-md bg-white p-8 rounded-xl shadow-sm border border-slate-200">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-slate-900">
            Create an account
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            Start managing your projects today
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Organization Name <span className="text-red-500">*</span>
            </label>
            <input
              {...register("orgName")}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
              placeholder="ABC Technologies"
            />
            {errors.orgName && (
              <p className="text-red-500 text-xs mt-1">
                {errors.orgName.message}
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Workspace Slug <span className="text-red-500">*</span>
            </label>
            <input
              {...register("slug")}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
              placeholder="abc-tech"
            />
            {errors.slug && (
              <p className="text-red-500 text-xs mt-1">{errors.slug.message}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Your Name <span className="text-red-500">*</span>
            </label>
            <input
              {...register("name")}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
              placeholder="John Doe"
            />
            {errors.name && (
              <p className="text-red-500 text-xs mt-1">{errors.name.message}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Email <span className="text-red-500">*</span>
            </label>
            <input
              {...register("email")}
              type="email"
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
              placeholder="you@company.com"
            />
            {errors.email && (
              <p className="text-red-500 text-xs mt-1">
                {errors.email.message}
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Password <span className="text-red-500">*</span>
            </label>
            <input
              {...register("password")}
              type="password"
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-brand-500"
              placeholder="••••••••"
            />
            {errors.password && (
              <p className="text-red-500 text-xs mt-1">
                {errors.password.message}
              </p>
            )}
            <p className="text-xs text-slate-500 mt-1">
              Already have an account (for example as a member of another
              workspace)? Enter your existing password to create your own
              organization with it.
            </p>
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-brand-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-brand-700 focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 disabled:opacity-70 transition-colors"
          >
            {isLoading ? "Creating Organization..." : "Create Organization"}
          </button>
        </form>

        <p className="text-center text-sm text-slate-500 mt-6">
          Already have an account?{" "}
          <Link
            to="/login"
            className="text-brand-600 font-medium hover:underline"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
