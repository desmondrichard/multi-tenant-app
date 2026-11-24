import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "./api";

// scope: "task" | "project" | "workspace"
// Opening a panel only asks for an EXISTING (cached) summary: it never calls the AI.
// The AI runs only when the user clicks "Generate" / "Regenerate".
export function useAISummary(scope, id) {
    const queryClient = useQueryClient();
    const path =
        scope === "workspace"
            ? "/ai/summarize/workspace"
            : `/ai/summarize/${scope}/${id}`;
    const key = ["ai-summary", scope, id || "all"];

    const query = useQuery({
        queryKey: key,
        queryFn: async () => (await api.post(path, { cachedOnly: true })).data,
        staleTime: 60 * 1000,
        enabled: scope === "workspace" || !!id,
    });

    const generate = useMutation({
        mutationFn: async (refresh = false) =>
            (await api.post(path, { refresh }, { timeout: 60000 })).data,
        onSuccess: (data) => {
            queryClient.setQueryData(key, data);
            if (data.empty) toast.info(data.message);
        },
        onError: (err) =>
            toast.error(
                err.code === "ECONNABORTED"
                    ? "The AI took too long to respond. Please try again."
                    : err.response?.data?.error || "Could not generate the summary",
            ),
    });

    return { query, generate, data: query.data };
}