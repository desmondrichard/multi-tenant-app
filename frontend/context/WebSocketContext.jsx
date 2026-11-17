import { createContext, useEffect, useRef } from "react";
import { useSelector } from "react-redux";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

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

    // Connect to WebSocket
    ws.current = new WebSocket(`ws://localhost:8080/api/ws?token=${token}`);

    ws.current.onopen = () => {
      console.log("✅ WebSocket Connected");
    };

    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "NEW_COMMENT") {
        // Instantly refetch comments and activities
        queryClient.invalidateQueries({ queryKey: ["comments", data.taskId] });
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
