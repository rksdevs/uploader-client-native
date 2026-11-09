import type { ButtonHTMLAttributes, ReactNode, SelectHTMLAttributes, InputHTMLAttributes } from "react";
import { AlertTriangle, ChevronDown, Crown, Info, Loader2, X } from "lucide-react";

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";

const BUTTON_VARIANTS: Record<ButtonVariant, string> = {
  primary: "bg-brand text-white hover:bg-brand-hover",
  secondary: "bg-muted text-foreground hover:bg-accent",
  outline: "border border-border bg-transparent text-foreground hover:bg-muted",
  ghost: "bg-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
  destructive: "border border-destructive/50 bg-transparent text-destructive hover:bg-destructive/10",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: "sm" | "md";
  busy?: boolean;
}

export function Button({ variant = "secondary", size = "md", busy, className, children, disabled, ...rest }: ButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled || busy}
      className={cx(
        // Tailwind v4 leaves buttons on the default arrow, so every button says it's clickable here.
        "inline-flex cursor-pointer items-center justify-center gap-2 rounded-control font-medium whitespace-nowrap transition-colors",
        "disabled:cursor-not-allowed disabled:opacity-50",
        size === "sm" ? "h-8 px-3 text-xs" : "h-9 px-4 text-sm",
        BUTTON_VARIANTS[variant],
        className,
      )}
      {...rest}
    >
      {busy && <Loader2 className="size-4 animate-spin" aria-hidden />}
      {children}
    </button>
  );
}

export function Card({ title, icon, action, children, className }: {
  title?: ReactNode;
  icon?: ReactNode;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <section className={cx("rounded-card border border-border bg-card", className)}>
      {title && (
        <header className="flex min-h-11 items-center justify-between gap-3 border-b border-border px-4 py-2">
          <h2 className="flex min-w-0 items-center gap-2 text-sm font-semibold">
            {icon}
            <span className="truncate">{title}</span>
          </h2>
          {action}
        </header>
      )}
      {children !== undefined && <div className="p-4">{children}</div>}
    </section>
  );
}

export type Tone = "neutral" | "brand" | "success" | "warning" | "danger" | "gold";

const BADGE_TONES: Record<Tone, string> = {
  neutral: "border-border text-muted-foreground",
  brand: "border-brand-text/40 text-brand-text",
  success: "border-success/40 text-success",
  warning: "border-warning/40 text-warning",
  danger: "border-destructive/40 text-destructive",
  gold: "border-gold/50 text-gold",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span className={cx("inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-semibold whitespace-nowrap", BADGE_TONES[tone])}>
      {children}
    </span>
  );
}

export function PremiumBadge() {
  return (
    <Badge tone="gold">
      <Crown className="size-3" aria-hidden /> Premium
    </Badge>
  );
}

export function Switch({ checked, onChange, disabled, label }: {
  checked: boolean;
  onChange: (next: boolean) => void;
  disabled?: boolean;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cx(
        "relative h-5 w-9 shrink-0 cursor-pointer rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "bg-brand" : "bg-muted-foreground/40",
      )}
    >
      <span className={cx("absolute top-0.5 left-0.5 size-4 rounded-full bg-white transition-transform", checked && "translate-x-4")} />
    </button>
  );
}

export function Select({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className={cx("relative", className)}>
      <select
        className="h-9 w-full appearance-none rounded-control border border-border bg-input py-0 pr-8 pl-3 text-sm text-foreground disabled:opacity-50 [&>option]:bg-input"
        {...rest}
      >
        {children}
      </select>
      <ChevronDown className="pointer-events-none absolute top-2.5 right-2.5 size-4 text-muted-foreground" aria-hidden />
    </div>
  );
}

export function Input({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cx("h-9 w-full rounded-control border border-border bg-input px-3 text-sm text-foreground placeholder:text-muted-foreground disabled:opacity-50", className)}
      {...rest}
    />
  );
}

export function Callout({ tone = "info", children, action }: {
  tone?: "info" | "warning" | "danger";
  children: ReactNode;
  action?: ReactNode;
}) {
  const styles = {
    info: "border-brand-text/30 bg-accent/40",
    warning: "border-warning/40 bg-warning/10",
    danger: "border-destructive/40 bg-destructive/10",
  }[tone];
  const Icon = tone === "info" ? Info : AlertTriangle;
  return (
    <div className={cx("flex items-start gap-2 rounded-card border p-3 text-sm", styles)}>
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">{children}</div>
      {action}
    </div>
  );
}

export function PageHeader({ title, description, badge, action }: {
  title: string;
  description: string;
  badge?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <header className="mb-5 flex flex-wrap items-start justify-between gap-3 border-b border-border pb-4">
      <div className="min-w-0">
        <h1 className="flex flex-wrap items-center gap-2 text-2xl font-semibold tracking-tight">
          {title} {badge}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>
      {action}
    </header>
  );
}

export function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="truncate font-mono text-sm font-semibold">{value}</div>
    </div>
  );
}

export function Modal({ title, onClose, children, wide }: {
  title: string;
  onClose?: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" role="dialog" aria-modal="true" aria-label={title}>
      <div className={cx("flex max-h-full w-full flex-col rounded-card border border-border bg-card", wide ? "max-w-4xl" : "max-w-md")}>
        <header className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
          <h2 className="text-base font-semibold">{title}</h2>
          {onClose && (
            <button type="button" onClick={onClose} aria-label="Close" className="rounded-control p-1 text-muted-foreground hover:bg-muted">
              <X className="size-4" aria-hidden />
            </button>
          )}
        </header>
        <div className="min-h-0 overflow-y-auto p-4">{children}</div>
      </div>
    </div>
  );
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-card border border-dashed border-border p-8 text-center">
      <p className="font-medium">{title}</p>
      {children && <div className="mt-1 text-sm text-muted-foreground">{children}</div>}
    </div>
  );
}

export function Table({ columns, children }: { columns: string[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto rounded-card border border-border bg-card">
      <table className="w-full min-w-[640px] border-collapse text-sm">
        <thead>
          <tr className="border-b border-border text-left text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
            {columns.map((column) => (
              <th key={column} className="px-3 py-2 whitespace-nowrap">{column}</th>
            ))}
          </tr>
        </thead>
        <tbody className="[&>tr]:border-b [&>tr]:border-border [&>tr:last-child]:border-0 [&_td]:px-3 [&_td]:py-2">
          {children}
        </tbody>
      </table>
    </div>
  );
}
