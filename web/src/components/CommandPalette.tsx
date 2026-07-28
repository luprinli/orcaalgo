import { useState, useEffect, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useTheme } from "next-themes"
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
} from "./ui/command"
import { Dialog, DialogContent } from "./ui/dialog"

const pages = [
  { label: "Monitor", path: "/", keys: "m" },
  { label: "Execution", path: "/execution", keys: "e" },
  { label: "Backtest", path: "/backtest", keys: "b" },
  { label: "Strategies", path: "/strategies", keys: "s" },
  { label: "Charting", path: "/market-data", keys: "c" },
  { label: "Simulation", path: "/simulate", keys: "i" },
  { label: "Calibration", path: "/calibrate" },
  { label: "Attribution", path: "/attribution" },
  { label: "Optimization", path: "/backtest?view=optimize" },
  { label: "Accounts", path: "/accounts" },
  { label: "Prop Firms", path: "/propfirm" },
  { label: "Settings", path: "/settings" },
  { label: "Integrations", path: "/brokers" },
  { label: "Admin", path: "/admin" },
  { label: "Emergency", path: "/emergency" },
]

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const nav = useNavigate()
  const { setTheme } = useTheme()

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((o) => !o)
      }
    }
    document.addEventListener("keydown", down)
    return () => document.removeEventListener("keydown", down)
  }, [])

  const run = useCallback(
    (fn: () => void) => {
      setOpen(false)
      fn()
    },
    [],
  )

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="top-1/3 translate-y-0 overflow-hidden rounded-xl p-0 max-w-lg" showCloseButton={false}>
        <Command className="[&_[cmdk-group-heading]]:text-muted-foreground">
          <CommandInput placeholder="Type a command or search..." />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup heading="Pages">
              {pages.map((p) => (
                <CommandItem
                  key={p.path}
                  value={p.label}
                  onSelect={() => run(() => nav(p.path))}
                >
                  <span>{p.label}</span>
                  {p.keys && <span className="ml-auto text-xs tracking-widest text-muted-foreground">{p.keys}</span>}
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup heading="Theme">
              <CommandItem onSelect={() => run(() => setTheme("light"))}>Light Mode</CommandItem>
              <CommandItem onSelect={() => run(() => setTheme("dark"))}>Dark Mode</CommandItem>
              <CommandItem onSelect={() => run(() => setTheme("system"))}>System</CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </DialogContent>
    </Dialog>
  )
}
