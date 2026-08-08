import { useEffect, useMemo, useState, type FocusEvent } from "react";

interface DateDropdownInputProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}

interface DatePartInputProps {
  id: string;
  dateLabel: string;
  partLabel: "年" | "月" | "日";
  value: string;
  options: string[];
  maxLength: number;
  placeholder: string;
  onChange: (value: string) => void;
}

type DatePart = string;

const parseDate = (value: string): [DatePart, DatePart, DatePart] => {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return ["", "", ""];
  return [match[1], String(Number(match[2])), String(Number(match[3]))];
};

const daysInMonth = (year: number, month: number): number =>
  new Date(year, month, 0).getDate();

const numericOnly = (value: string, maxLength: number): string =>
  value.replace(/\D/g, "").slice(0, maxLength);

function DatePartInput({
  id,
  dateLabel,
  partLabel,
  value,
  options,
  maxLength,
  placeholder,
  onChange,
}: DatePartInputProps) {
  const [open, setOpen] = useState(false);

  const closeWhenFocusLeaves = (event: FocusEvent<HTMLDivElement>) => {
    const nextTarget = event.relatedTarget;
    if (nextTarget instanceof Node && event.currentTarget.contains(nextTarget)) return;
    setOpen(false);
  };

  return (
    <div className="date-part-field" onBlur={closeWhenFocusLeaves}>
      <span className="date-part-label">{partLabel}</span>
      <div className="date-part-control">
        <input
          id={`${id}-${partLabel}`}
          aria-label={`${dateLabel} ${partLabel}`}
          type="text"
          inputMode="numeric"
          autoComplete="off"
          placeholder={placeholder}
          value={value}
          maxLength={maxLength}
          onFocus={() => setOpen(true)}
          onChange={(event) => onChange(numericOnly(event.target.value, maxLength))}
          aria-expanded={open}
          aria-controls={`${id}-${partLabel}-options`}
          role="combobox"
        />
        <button
          className="date-part-toggle"
          type="button"
          aria-label={`${dateLabel} ${partLabel}の候補を表示`}
          aria-expanded={open}
          aria-controls={`${id}-${partLabel}-options`}
          onClick={() => setOpen((current) => !current)}
        >
          <span aria-hidden="true">⌄</span>
        </button>
      </div>
      {open && (
        <div
          id={`${id}-${partLabel}-options`}
          className="date-part-options"
          role="listbox"
          aria-label={`${dateLabel} ${partLabel}の候補`}
        >
          {options.map((option) => (
            <button
              key={option}
              type="button"
              role="option"
              aria-selected={value === option}
              className={value === option ? "is-selected" : undefined}
              onClick={() => {
                onChange(option);
                setOpen(false);
              }}
            >
              {option}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export function DateDropdownInput({ id, label, value, onChange }: DateDropdownInputProps) {
  const [parsedYear, parsedMonth, parsedDay] = useMemo(() => parseDate(value), [value]);
  const [year, setYear] = useState(parsedYear);
  const [month, setMonth] = useState(parsedMonth);
  const [day, setDay] = useState(parsedDay);
  const currentYear = new Date().getFullYear();

  const years = useMemo(
    () => Array.from({ length: currentYear - 1999 }, (_, index) => String(currentYear - index)),
    [currentYear],
  );
  const months = useMemo(() => Array.from({ length: 12 }, (_, index) => String(index + 1)), []);
  const days = useMemo(() => Array.from({ length: 31 }, (_, index) => String(index + 1)), []);

  useEffect(() => {
    const [nextYear, nextMonth, nextDay] = parseDate(value);
    setYear(nextYear);
    setMonth(nextMonth);
    setDay(nextDay);
  }, [value]);

  const publish = (nextYear: string, nextMonth: string, nextDay: string) => {
    if (!nextYear || !nextMonth || !nextDay) {
      onChange("");
      return;
    }

    const yearNumber = Number(nextYear);
    const monthNumber = Number(nextMonth);
    const dayNumber = Number(nextDay);
    if (
      !Number.isInteger(yearNumber) ||
      !Number.isInteger(monthNumber) ||
      !Number.isInteger(dayNumber) ||
      yearNumber < 2000 ||
      yearNumber > currentYear ||
      monthNumber < 1 ||
      monthNumber > 12
    ) {
      onChange("");
      return;
    }

    const maxDay = daysInMonth(yearNumber, monthNumber);
    if (dayNumber < 1 || dayNumber > maxDay) {
      onChange("");
      return;
    }

    onChange(
      `${String(yearNumber).padStart(4, "0")}-${String(monthNumber).padStart(2, "0")}-${String(dayNumber).padStart(2, "0")}`,
    );
  };

  const updateYear = (next: string) => {
    setYear(next);
    publish(next, month, day);
  };
  const updateMonth = (next: string) => {
    setMonth(next);
    publish(year, next, day);
  };
  const updateDay = (next: string) => {
    setDay(next);
    publish(year, month, next);
  };

  return (
    <fieldset className="date-dropdown" aria-labelledby={`${id}-legend`}>
      <legend id={`${id}-legend`}>{label}</legend>
      <div className="date-dropdown-fields">
        <DatePartInput
          id={id}
          dateLabel={label}
          partLabel="年"
          value={year}
          options={years}
          maxLength={4}
          placeholder="YYYY"
          onChange={updateYear}
        />
        <DatePartInput
          id={id}
          dateLabel={label}
          partLabel="月"
          value={month}
          options={months}
          maxLength={2}
          placeholder="MM"
          onChange={updateMonth}
        />
        <DatePartInput
          id={id}
          dateLabel={label}
          partLabel="日"
          value={day}
          options={days}
          maxLength={2}
          placeholder="DD"
          onChange={updateDay}
        />
      </div>
    </fieldset>
  );
}
