import { createContext, useEffect, useRef } from "react";
import { useSelector } from "react-redux";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "../lib/api";
import { forceLogout } from "../lib/session";

export const WebSocketContext = createContext(null);

export function WebSocketProvider({ children }) {
  const { token, isAuthenticated } = useSelector((state) => state.auth);
  const queryClient = useQueryClient();
  const ws = useRef(null);

  useEffect(() => {
    if (!isAuthenticated || !token) return;

    let closedByUs = false;
    let retryTimer = null;

    // Determine the correct WebSocket URL (http -> ws, https -> wss)
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8080/api";
    const wsUrl = apiUrl.replace(/^http/, "ws");

    const handleMessage = (data) => {
      switch (data.type) {
        // The owner/admin removed THIS user: log out immediately
        case "FORCE_LOGOUT":
          forceLogout("removed");
          break;

        // This user's role was changed: reload the profile
        case "ROLE_CHANGED":
          queryClient.invalidateQueries({ queryKey: ["me"] });
          toast.info("Your role in this workspace was updated.");
          break;

        // A notification for THIS user: refresh the bell and show a toast
        case "NOTIFICATION":
          queryClient.invalidateQueries({ queryKey: ["notifications"] });
          toast.info(data.title || "New notification", {
            description: data.message,
          });
          break;

        case "NEW_COMMENT":
          if (data.subtaskId) {
            queryClient.invalidateQueries({
              queryKey: ["subtask-comments", data.taskId, data.subtaskId],
            });
          } else {
            queryClient.invalidateQueries({
              queryKey: ["comments", data.taskId],
            });
          }
          queryClient.invalidateQueries({
            queryKey: ["activities", data.taskId],
          });
          queryClient.invalidateQueries({ queryKey: ["activity-feed"] });
          break;

        case "TASKS_UPDATED":
          queryClient.invalidateQueries({ queryKey: ["activity-feed"] });
          queryClient.invalidateQueries({ queryKey: ["tasks"] });
          queryClient.invalidateQueries({ queryKey: ["analytics"] });
          queryClient.invalidateQueries({ queryKey: ["my-tasks"] });
          break;

        case "PROJECTS_UPDATED":
          queryClient.invalidateQueries({ queryKey: ["activity-feed"] });
          queryClient.invalidateQueries({ queryKey: ["projects"] });
          queryClient.invalidateQueries({ queryKey: ["tasks"] });
          queryClient.invalidateQueries({ queryKey: ["analytics"] });
          queryClient.invalidateQueries({ queryKey: ["my-tasks"] });
          break;

        default:
          break;
      }
    };

    const connect = () => {
      const socket = new WebSocket(`${wsUrl}/ws?token=${token}`);
      ws.current = socket;
      let opened = false;

      socket.onopen = () => {
        opened = true;
        console.log("✅ WebSocket Connected");
      };

      socket.onmessage = (event) => {
        try {
          handleMessage(JSON.parse(event.data));
        } catch (e) {
          console.error("Bad WebSocket message", e);
        }
      };

      socket.onclose = () => {
        console.log("❌ WebSocket Disconnected");
        if (closedByUs) return;

        // The server refuses the connection when the token is no longer valid
        // (e.g. user removed). Ask /me: a 401 there logs the user out.
        if (!opened) api.get("/me").catch(() => {});

        // Reconnect after 3 seconds
        retryTimer = setTimeout(connect, 3000);
      };
    };

    connect();

    return () => {
      closedByUs = true;
      clearTimeout(retryTimer);
      if (ws.current) ws.current.close();
    };
  }, [isAuthenticated, token, queryClient]);

  return (
    <WebSocketContext.Provider value={ws.current}>
      {children}
    </WebSocketContext.Provider>
  );
}
