import { createContext, useEffect, useRef } from "react";
import { useSelector } from "react-redux";
import { useQueryClient } from "@tanstack/react-query";

export const WebSocketContext = createContext(null);

export function WebSocketProvider({ children }) {
  const { token, isAuthenticated } = useSelector((state) => state.auth);
  const queryClient = useQueryClient();
  const ws = useRef(null);

  useEffect(() => {
    if (!isAuthenticated || !token) {
      if (ws.current) ws.current.close();
      return;
    }

    // 1. Determine the correct WebSocket URL
    // If we are in production, replace http/https with ws/wss
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8080/api";
    const wsUrl = apiUrl.replace(/^http/, "ws"); // turns http:// into ws://, https:// into wss://

    // 2. Connect
    ws.current = new WebSocket(`${wsUrl}/ws?token=${token}`);

    ws.current.onopen = () => {
      console.log("✅ WebSocket Connected");
    };

    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "NEW_COMMENT") {
        console.log("📩 Real-time update received!");
        // Instantly refetch comments and activities
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
      }
    };

    ws.current.onclose = () => {
      console.log("❌ WebSocket Disconnected");
    };

    return () => {
      if (ws.current) ws.current.close();
    };
  }, [isAuthenticated, token, queryClient]);

  return (
    <WebSocketContext.Provider value={ws.current}>
      {children}
    </WebSocketContext.Provider>
  );
}
