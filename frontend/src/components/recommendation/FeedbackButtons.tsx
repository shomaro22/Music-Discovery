import type { FeedbackStatus, FeedbackType } from "../../types/feedback";

const OPTIONS: ReadonlyArray<{ type: FeedbackType; label: string }> = [
  { type: "LIKE", label: "好き" },
  { type: "MAYBE", label: "もう少し聴きたい" },
  { type: "SKIP", label: "今回は違う" },
];

interface FeedbackButtonsProps {
  current: FeedbackStatus;
  disabled: boolean;
  error: string;
  onSelect: (feedbackType: FeedbackType) => void;
}

export function FeedbackButtons({ current, disabled, error, onSelect }: FeedbackButtonsProps) {
  return (
    <fieldset className="feedback-actions" disabled={disabled}>
      <legend>この推薦はどうでしたか？</legend>
      <div className="feedback-button-row">
        {OPTIONS.map((option) => (
          <button
            aria-label={`${option.label} ${option.type}`}
            key={option.type}
            type="button"
            className={current === option.type ? "feedback-button selected" : "feedback-button"}
            aria-pressed={current === option.type}
            onClick={() => onSelect(option.type)}
          >
            <span>{option.label}</span>
            <small>{option.type}</small>
          </button>
        ))}
      </div>
      {disabled && <p className="feedback-status-message" aria-live="polite">保存中...</p>}
      {error && <p className="feedback-error" role="alert">{error}</p>}
    </fieldset>
  );
}
