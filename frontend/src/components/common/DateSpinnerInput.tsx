import { useEffect, useMemo, useState } from "react";

interface DateSpinnerInputProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}

const parseDate = (value: string): [number | "", number | "", number | ""] => {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return ["", "", ""];
  return [Number(match[1]), Number(match[2]), Number(match[3])];
};

const daysInMonth = (year: number, month: number): number =>
  new Date(year, month, 0).getDate();

export function DateSpinnerInput({ id, label, value, onChange }: DateSpinnerInputProps) {
  const [initialYear, initialMonth, initialDay] = useMemo(() => parseDate(value), [value]);
  const [year, setYear] = useState<number | "">(initialYear);
  const [month, setMonth] = useState<number | "">(initialMonth);
  const [day, setDay] = useState<number | "">(initialDay);
  const currentYear = new Date().getFullYear();

  useEffect(() => {
    const [nextYear, nextMonth, nextDay] = parseDate(value);
    setYear(nextYear);
    setMonth(nextMonth);
    setDay(nextDay);
  }, [value]);

  const publish = (nextYear: number | "", nextMonth: number | "", nextDay: number | "") => {
    if (nextYear === "" || nextMonth === "" || nextDay === "") {
      onChange("");
      return;
    }
    const safeDay = Math.min(nextDay, daysInMonth(nextYear, nextMonth));
    onChange(`${String(nextYear).padStart(4, "0")}-${String(nextMonth).padStart(2, "0")}-${String(safeDay).padStart(2, "0")}`);
  };

  return (
    <fieldset className="date-spinner" aria-labelledby={`${id}-legend`}>
      <legend id={`${id}-legend`}>{label}</legend>
      <div className="date-spinner-fields">
        <label>
          <span>年</span>
          <input
            aria-label={`${label} 年`}
            type="number"
            inputMode="numeric"
            min={2000}
            max={currentYear + 1}
            step={1}
            value={year}
            onChange={(event) => {
              const next = event.target.value === "" ? "" : Number(event.target.value);
              setYear(next);
              publish(next, month, day);
            }}
          />
        </label>
        <label>
          <span>月</span>
          <input
            aria-label={`${label} 月`}
            type="number"
            inputMode="numeric"
            min={1}
            max={12}
            step={1}
            value={month}
            onChange={(event) => {
              const next = event.target.value === "" ? "" : Math.min(12, Math.max(1, Number(event.target.value)));
              setMonth(next);
              publish(year, next, day);
            }}
          />
        </label>
        <label>
          <span>日</span>
          <input
            aria-label={`${label} 日`}
            type="number"
            inputMode="numeric"
            min={1}
            max={year !== "" && month !== "" ? daysInMonth(year, month) : 31}
            step={1}
            value={day}
            onChange={(event) => {
              const maxDay = year !== "" && month !== "" ? daysInMonth(year, month) : 31;
              const next = event.target.value === "" ? "" : Math.min(maxDay, Math.max(1, Number(event.target.value)));
              setDay(next);
              publish(year, month, next);
            }}
          />
        </label>
      </div>
      {value && <button className="date-spinner-clear" type="button" onClick={() => onChange("")}>日付をクリア</button>}
    </fieldset>
  );
}
