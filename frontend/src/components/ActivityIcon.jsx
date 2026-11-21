import {
  CheckCircle2,
  MessageSquare,
  Plus,
  ArrowRightLeft,
  Trash2,
  Pencil,
} from "lucide-react";

const STYLES = {
  done: { box: "bg-green-100 text-green-600", Icon: CheckCircle2 },
  comment: { box: "bg-purple-100 text-purple-600", Icon: MessageSquare },
  created: { box: "bg-blue-100 text-blue-600", Icon: Plus },
  status: { box: "bg-amber-100 text-amber-600", Icon: ArrowRightLeft },
  deleted: { box: "bg-red-100 text-red-600", Icon: Trash2 },
  edited: { box: "bg-indigo-100 text-indigo-600", Icon: Pencil },
};

export default function ActivityIcon({ type, className = "" }) {
  const { box, Icon } = STYLES[type] || STYLES.created;
  return (
    <div
      className={`w-6 h-6 rounded-full flex items-center justify-center border-2 border-white shadow-sm ${box} ${className}`}
    >
      <Icon className="w-3.5 h-3.5" />
    </div>
  );
}