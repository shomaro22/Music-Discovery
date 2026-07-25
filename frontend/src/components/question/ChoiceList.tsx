import type { Question } from "../../types/question";

interface ChoiceListProps { question: Question; selectedIds: string[]; onChange: (ids: string[]) => void; }

export function ChoiceList({ question, selectedIds, onChange }: ChoiceListProps) {
  const toggle = (choiceId: string): void => {
    if (question.answer_type === "single") { onChange([choiceId]); return; }
    if (selectedIds.includes(choiceId)) { onChange(selectedIds.filter((id) => id !== choiceId)); return; }
    if (question.max_selections !== null && selectedIds.length >= question.max_selections) return;
    onChange([...selectedIds, choiceId]);
  };
  return <div className="choice-list" role={question.answer_type === "single" ? "radiogroup" : "group"} aria-label={question.title}>
    {question.choices.map((choice) => {
      const selected = selectedIds.includes(choice.id);
      const disabled = !selected && question.answer_type === "multiple" && question.max_selections !== null && selectedIds.length >= question.max_selections;
      return <button key={choice.id} type="button" role={question.answer_type === "single" ? "radio" : "checkbox"} aria-checked={selected} className={selected ? "choice-button selected" : "choice-button"} disabled={disabled} onClick={() => toggle(choice.id)}>{choice.label}</button>;
    })}
  </div>;
}
