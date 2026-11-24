import { Sparkles, RefreshCw, Copy, AlertTriangle, ArrowRight } from "lucide-react";
import { toast } from "sonner";
import { useAISummary } from "../lib/useAISummary";
import { timeAgo } from "../lib/activity";

function Section({ title, items, tone }) {
  if (!items || items.length === 0) return null;
  const color =
    tone === "bad"
      ? "text-red-700 bg-red-50 border-red-100"
      : tone === "good"
        ? "text-green-700 bg-green-50 border-green-100"
        : "text-slate-700 bg-slate-50 border-slate-100";
  return (
    <div className={`rounded-lg border p-3 ${color}`}>
      <h4 className="text-xs font-semibold uppercase tracking-wide mb-2 flex items-center gap-1.5">
        {tone === "bad" && <AlertTriangle className="w-3.5 h-3.5" />}
        {tone === "good" && <ArrowRight className="w-3.5 h-3.5" />}
        {title}
      </h4>
      <ul className="space-y-1 text-sm">
        {items.map((item, i) => (
          <li key={i} className="flex gap-2">
            <span>•</span>
            <span>{item}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

// Shows an AI summary for a task, a project or the whole workspace.
export default function AISummaryPanel({ scope, id, emptyHint, buttonLabel }) {
  const { query, generate, data } = useAISummary(scope, id);
  const summary = data?.summary;
  const isGenerating = generate.isPending;

  const copy = () => {
    const text = [
      summary.overview,
      summary.progress,
      summary.keyPoints.length ? "Key points:\n- " + summary.keyPoints.join("\n- ") : "",
      summary.blockers.length ? "Blockers:\n- " + summary.blockers.join("\n- ") : "",
      summary.nextSteps.length ? "Next steps:\n- " + summary.nextSteps.join("\n- ") : "",
    ]
      .filter(Boolean)
      .join("\n\n");
    navigator.clipboard.writeText(text).then(
      () => toast.success("Summary copied"),
      () => toast.error("Could not copy"),
    );
  };

  // 1. checking for an existing summary
  if (query.isLoading) {
    return <p className="text-sm text-slate-400">Checking for a saved summary...</p>;
  }

  // 2. generating
  if (isGenerating && !summary) {
    return (
      <div className="flex items-center gap-3 text-sm text-slate-500 py-6">
        <Sparkles className="w-5 h-5 text-brand-500 animate-pulse" />
        Reading the data and writing the summary (5-15 sec)...
      </div>
    );
  }

  // 3. nothing yet: invite the user to generate one
  if (!summary) {
    return (
      <div className="text-center py-6">
        <Sparkles className="w-8 h-8 text-brand-500 mx-auto mb-3" />
        <p className="text-sm text-slate-600 mb-1">
          {data?.empty ? data.message : emptyHint || "No summary yet."}
        </p>
        {!data?.empty && (
          <p className="text-xs text-slate-400 mb-4">
            It is created only when you click the button.
          </p>
        )}
        <button
          onClick={() => generate.mutate(false)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors"
        >
          <Sparkles className="w-4 h-4" />
          {buttonLabel || "Generate summary"}
        </button>
        {query.isError && (
          <p className="text-xs text-red-500 mt-3">
            {query.error?.response?.data?.error || "Could not load the summary"}
          </p>
        )}
      </div>
    );
  }

  // 4. the summary
  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium text-slate-900">{summary.overview}</p>
        {summary.progress && (
          <p className="text-sm text-slate-600 mt-1">{summary.progress}</p>
        )}
      </div>

      <Section title="Key points" items={summary.keyPoints} />
      <Section title="Blockers" items={summary.blockers} tone="bad" />
      <Section title="Next steps" items={summary.nextSteps} tone="good" />

      <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
        <span className="text-[11px] text-slate-400">
          AI-generated {data.generatedAt ? timeAgo(data.generatedAt) : ""}
          {data.cached ? " (saved)" : ""}. It can make mistakes.
        </span>
        <div className="flex gap-2">
          <button
            onClick={copy}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-600 border border-slate-200 rounded-lg hover:bg-slate-50"
          >
            <Copy className="w-3.5 h-3.5" /> Copy
          </button>
          <button
            onClick={() => generate.mutate(true)}
            disabled={isGenerating}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-brand-700 border border-brand-200 rounded-lg hover:bg-brand-50 disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isGenerating ? "animate-spin" : ""}`} />
            {isGenerating ? "Updating..." : "Regenerate"}
          </button>
        </div>
      </div>
    </div>
  );
}