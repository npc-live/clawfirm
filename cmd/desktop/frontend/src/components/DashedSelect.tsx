import { useEffect, useRef, useState } from "react";

export function DashedSelect({ value, onChange, options, placeholder }: {
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
  placeholder?: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false); };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const selected = options.find(o => o.value === value);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-2 py-1.5 text-[11px] font-mono border border-dashed border-[rgba(30,28,23,0.25)] hover:border-[rgba(30,28,23,0.4)] bg-[rgba(30,28,23,0.03)] text-[#1e1c17] transition-colors text-left"
      >
        <span className={selected ? "" : "text-[rgba(30,28,23,0.25)]"}>
          {selected ? selected.label : (placeholder ?? "-- select --")}
        </span>
        <span className="text-[rgba(30,28,23,0.3)] text-[9px] ml-2">{open ? "▲" : "▼"}</span>
      </button>
      {open && (
        <div className="absolute z-50 left-0 right-0 mt-0.5 border border-dashed border-[rgba(30,28,23,0.3)] bg-[#ece8df] max-h-40 overflow-y-auto">
          {options.map(o => (
            <button
              key={o.value}
              type="button"
              onClick={() => { onChange(o.value); setOpen(false); }}
              className={`w-full text-left px-2 py-1.5 text-[11px] font-mono hover:bg-[rgba(30,28,23,0.08)] transition-colors ${
                o.value === value ? "text-[#1e1c17] bg-[rgba(30,28,23,0.06)]" : "text-[rgba(30,28,23,0.6)]"
              }`}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
