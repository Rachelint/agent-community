import type { Label } from '../../api/types';
import { labelTextColor } from './palette';

export function LabelBadge({
  label,
  onRemove,
}: {
  label: Label;
  onRemove?: () => void;
}) {
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold"
      style={{
        backgroundColor: `#${label.color}`,
        color: labelTextColor(label.color),
      }}
      title={label.description || label.name}
    >
      {label.name}
      {onRemove ? (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="text-[11px] leading-none opacity-80 hover:opacity-100"
          aria-label={`remove ${label.name}`}
        >
          ×
        </button>
      ) : null}
    </span>
  );
}
