import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../lib/api";
import { MessageSquare, X, Send, Sparkles } from "lucide-react";

export default function AIChatbot() {
  const [isOpen, setIsOpen] = useState(false);
  const [question, setQuestion] = useState("");
  const [messages, setMessages] = useState([
    {
      role: "ai",
      text: "Hi! I am your workspace assistant. Ask me anything about your projects or tasks.",
    },
  ]);

  const askMutation = useMutation({
    mutationFn: (q) => api.post("/ai/ask", { question: q }),
    onSuccess: (res) => {
      setMessages((prev) => [...prev, { role: "ai", text: res.data.answer }]);
    },
    onError: () => {
      setMessages((prev) => [
        ...prev,
        {
          role: "ai",
          text: "Sorry, I am having trouble connecting right now. Please try again.",
        },
      ]);
    },
  });

  const handleSend = (e) => {
    e.preventDefault();
    if (!question.trim()) return;

    // Add user message to UI
    setMessages((prev) => [...prev, { role: "user", text: question }]);
    askMutation.mutate(question);
    setQuestion("");
  };

  return (
    <div className="fixed bottom-6 right-6 z-50">
      {/* Chat Window */}
      {isOpen && (
        <div className="bg-white w-80 h-96 rounded-xl shadow-2xl border border-slate-200 flex flex-col mb-4 animate-in slide-in-from-bottom-5">
          <div className="bg-brand-600 text-white p-4 rounded-t-xl flex justify-between items-center">
            <h3 className="font-bold flex items-center gap-2">
              <Sparkles className="w-4 h-4" /> AI Assistant
            </h3>
            <button
              onClick={() => setIsOpen(false)}
              className="hover:bg-brand-700 p-1 rounded"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          <div className="flex-1 p-4 overflow-y-auto space-y-3 bg-slate-50">
            {messages.map((msg, i) => (
              <div
                key={i}
                className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`max-w-[80%] p-2 rounded-lg text-sm ${msg.role === "user" ? "bg-brand-600 text-white rounded-br-none" : "bg-white border border-slate-200 text-slate-700 rounded-bl-none shadow-sm"}`}
                >
                  {msg.text}
                </div>
              </div>
            ))}
            {askMutation.isPending && (
              <div className="text-xs text-slate-400 italic">
                AI is thinking... (may take a minute on free tier)
              </div>
            )}
          </div>

          <form
            onSubmit={handleSend}
            className="p-3 border-t border-slate-200 bg-white rounded-b-xl flex gap-2"
          >
            <input
              type="text"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="Ask about tasks, projects..."
              className="flex-1 px-3 py-1.5 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
            />
            <button
              type="submit"
              disabled={askMutation.isPending || !question.trim()}
              className="bg-brand-600 text-white p-2 rounded-lg hover:bg-brand-700 disabled:opacity-50"
            >
              <Send className="w-4 h-4" />
            </button>
          </form>
        </div>
      )}

      {/* Floating Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="bg-brand-600 text-white w-14 h-14 rounded-full shadow-lg flex items-center justify-center hover:bg-brand-700 transition-transform hover:scale-105"
      >
        <MessageSquare className="w-6 h-6" />
      </button>
    </div>
  );
}
