import { useState, useRef, useEffect } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../lib/api";
import { MessageSquare, X, Send, Sparkles, RotateCcw } from "lucide-react";

const GREETING = {
  role: "ai",
  local: true, // not sent back to the server as conversation history
  text: "Hi! I'm your workspace assistant. Ask me about your team, roles, projects, tasks or recent activity.",
};

const SUGGESTIONS = [
  "How many team members do we have?",
  "Who is the owner?",
  "What is my role?",
  "Which tasks are overdue?",
  "What tasks are assigned to me?",
  "Summarize project progress",
];

// Shows **bold** and "- " bullet lines nicely (no extra library needed)
function RichText({ text }) {
  const inline = (line) =>
    line
      .split(/(\*\*[^*]+\*\*)/g)
      .map((part, i) =>
        part.startsWith("**") && part.endsWith("**") ? (
          <strong key={i}>{part.slice(2, -2)}</strong>
        ) : (
          part
        ),
      );

  return (
    <div className="space-y-1">
      {text.split("\n").map((line, i) => {
        const t = line.trim();
        if (!t) return null;
        const bullet = t.match(/^[-*•]\s+(.*)/);
        if (bullet) {
          return (
            <div key={i} className="flex gap-2">
              <span>•</span>
              <span>{inline(bullet[1])}</span>
            </div>
          );
        }
        return <p key={i}>{inline(t)}</p>;
      })}
    </div>
  );
}

export default function AIChatbot() {
  const [isOpen, setIsOpen] = useState(false);
  const [question, setQuestion] = useState("");
  const [messages, setMessages] = useState([GREETING]);
  const bottomRef = useRef(null);

  // Always scroll to the newest message
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, isOpen]);

  const askMutation = useMutation({
    mutationFn: ({ q, history }) =>
      api.post("/ai/ask", { question: q, history }, { timeout: 60000 }),
    onSuccess: (res) => {
      setMessages((prev) => [...prev, { role: "ai", text: res.data.answer }]);
    },
    onError: (err) => {
      setMessages((prev) => [
        ...prev,
        {
          role: "ai",
          local: true,
          text:
            err.code === "ECONNABORTED"
              ? "That took too long. Please try again."
              : err.response?.data?.error ||
                "Sorry, I couldn't answer right now. Please try again.",
        },
      ]);
    },
  });

  const send = (text) => {
    const q = text.trim();
    if (!q || askMutation.isPending) return;

    // Send the last few real messages so follow-ups ("what is his role?") work
    const history = messages
      .filter((m) => !m.local)
      .slice(-6)
      .map(({ role, text }) => ({ role, text }));

    setMessages((prev) => [...prev, { role: "user", text: q }]);
    askMutation.mutate({ q, history });
    setQuestion("");
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    send(question);
  };

  const onlyGreeting = messages.length === 1;

  return (
    <div className="fixed bottom-6 right-6 z-50">
      {/* Chat Window */}
      {isOpen && (
        <div className="bg-white w-96 max-w-[calc(100vw-3rem)] h-128 max-h-[75vh] rounded-xl shadow-2xl border border-slate-200 flex flex-col mb-4">
          <div className="bg-brand-600 text-white p-4 rounded-t-xl flex justify-between items-center">
            <h3 className="font-bold flex items-center gap-2">
              <Sparkles className="w-4 h-4" /> Workspace Assistant
            </h3>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setMessages([GREETING])}
                title="Clear chat"
                className="hover:bg-brand-700 p-1 rounded"
              >
                <RotateCcw className="w-4 h-4" />
              </button>
              <button
                onClick={() => setIsOpen(false)}
                className="hover:bg-brand-700 p-1 rounded"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div className="flex-1 p-4 overflow-y-auto space-y-3 bg-slate-50">
            {messages.map((msg, i) => (
              <div
                key={i}
                className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`max-w-[85%] px-3 py-2 rounded-lg text-sm ${
                    msg.role === "user"
                      ? "bg-brand-600 text-white rounded-br-none"
                      : "bg-white border border-slate-200 text-slate-700 rounded-bl-none shadow-sm"
                  }`}
                >
                  {msg.role === "user" ? (
                    msg.text
                  ) : (
                    <RichText text={msg.text} />
                  )}
                </div>
              </div>
            ))}

            {/* Suggested questions */}
            {onlyGreeting && (
              <div className="flex flex-wrap gap-2 pt-1">
                {SUGGESTIONS.map((s) => (
                  <button
                    key={s}
                    onClick={() => send(s)}
                    className="text-xs px-3 py-1.5 rounded-full border border-brand-200 text-brand-700 bg-white hover:bg-brand-50 transition-colors"
                  >
                    {s}
                  </button>
                ))}
              </div>
            )}

            {askMutation.isPending && (
              <div className="text-xs text-slate-400 italic">Thinking...</div>
            )}
            <div ref={bottomRef} />
          </div>

          <form
            onSubmit={handleSubmit}
            className="p-3 border-t border-slate-200 bg-white rounded-b-xl flex gap-2"
          >
            <div className="relative flex-1">
              <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                maxLength={500}
                placeholder="Ask about your team, tasks, projects..."
                className="w-full pr-7 px-3 py-1.5 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
              />
              {!question.trim() && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-red-500 text-sm pointer-events-none">
                  *
                </span>
              )}
            </div>
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
        className="bg-brand-600 text-white w-14 h-14 rounded-full shadow-lg flex items-center justify-center hover:bg-brand-700 transition-transform hover:scale-105 ml-auto block"
      >
        <MessageSquare className="w-6 h-6 mx-auto" />
      </button>
    </div>
  );
}
