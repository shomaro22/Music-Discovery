export function QuestionProgress({ current, total }: { current: number; total: number }) {
  return <div className="question-progress" aria-label={`質問 ${current} / ${total}`}><span>{current} / {total}</span><div><span style={{ width: `${(current / total) * 100}%` }} /></div></div>;
}
