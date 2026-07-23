<div align="center">

Topics: metatrader5, mql5, prop-firm, expert-advisor, mql4, metatrader, forex-trading, automated-trading, risk-management, multi-account, prop-trading, copy-trading, drawdown-protection, mt4, mt5, trading-bot, mt5-kill-switch, prop-firm-risk-engine, multi-account-drawdown-guard

# Prop Matrix Engine

**A multi-account risk command center for prop trading and MetaTrader 5 style terminal fleets. It tracks account equity, drawdown, exposure, command acknowledgements, chart-line copy events, and global kill-switch actions from one Windows desktop dashboard.**

<br>

[![Stars](https://img.shields.io/badge/Stars-Repository-00D4AA?style=for-the-badge)](https://github.com/your-username/volume-profile-mt5/stargazers)
[![Forks](https://img.shields.io/badge/Forks-Community-4D9FFF?style=for-the-badge)](https://github.com/your-username/volume-profile-mt5/network)
[![Issues](https://img.shields.io/badge/Issues-Tracker-FF4D6A?style=for-the-badge)](https://github.com/your-username/volume-profile-mt5/issues)
[![Platform](https://img.shields.io/badge/Platform-MetaTrader%205-00D4AA?style=for-the-badge)](https://www.metatrader5.com)
[![License](https://img.shields.io/badge/License-MIT-4D9FFF?style=for-the-badge)](LICENSE)
</div>

---

## Screenshot

<img width="1251" height="737" alt="Screenshot_1" src="https://github.com/user-attachments/assets/2af2aa33-e133-44a4-8029-7869f379e6bc" />

---

## 🎬 Demo

<div align="center">

<img src="https://i.imgur.com/JA3gptn.gif" alt="Demo">

</div>






---

## Why This Project

Prop firm trading requires strict control over drawdown, exposure, account consistency, and emergency shutdown behavior. **Prop Matrix Engine** models that control layer as a dedicated operator dashboard, with clear account tables, order command history, risk thresholds, and a global kill switch.

It is useful for:

- Prop trading risk-control prototypes
- Multi-account MT5 monitoring tools
- Trade copier command center concepts
- SignalR terminal integration experiments
- Risk engine and kill-switch UI portfolios

---

## What It Does

| Module | Description |
|---|---|
| Account Registry | Tracks connected terminal clients and latest account snapshots |
| Risk Engine | Aggregates global equity and evaluates drawdown thresholds |
| Kill Switch | Dispatches close positions, cancel pending orders, and disable trading commands |
| Chart-Line Copy | Converts chart-line events into replicated pending orders |
| Account Grid | Displays balance, equity, floating P/L, exposure, status, and last tick |
| Command Grid | Logs trade command results with timing and success status |
| Event Log | Shows operator and system activity in real time |

---

## Feature Highlights

| Feature | Detail |
|---|---|
| Multi-Account Dashboard | Add multiple MT5-style accounts and watch them together |
| Global Drawdown Guard | Tracks total equity against daily peak equity |
| Pre-Breach Buffer | Fires before the formal drawdown limit is fully reached |
| Auto Kill Switch | Optional automatic shutdown when risk limits are hit |
| Manual Kill Switch | One button to close positions, cancel pending orders, and disable trading |
| Chart-Line Replication | Turns source account chart lines into target pending orders |
| SignalR Client Mode | Includes a real-time hub client for terminal integrations |
| Simulation Mode | Ships with a simulated MT5 client for demos and UI testing |

---

## Risk Engine

```text
Account snapshots
   |
   v
Total equity + daily peak equity
   |
   v
Drawdown percentage
   |
   +-- Below threshold      -> Risk Engine: Armed
   |
   +-- At pre-breach level  -> KillSwitchRequired event
```

Default risk controls:

| Control | Default |
|---|---|
| Daily drawdown limit | 5.00% |
| Pre-breach buffer | 0.50% |
| Copy lot size | 0.10 lots |
| Auto kill switch | Enabled |
| Copy chart lines | Enabled |

---

## Terminal Clients

The engine supports two terminal client styles:

```text
SignalR Client
  - Connects to a remote hub
  - Receives AccountSnapshot events
  - Receives ChartLineChanged events
  - Sends trade commands

Simulated Client
  - Generates equity drift
  - Emits chart-line events
  - Accepts kill-switch commands
  - Useful for demos and screenshots
```

---

## Command Sequence

When the global kill switch fires, the registry dispatches:

```text
1. CloseAllPositions
2. CancelAllPendingOrders
3. DisableTrading
```

Each response is tracked in the command grid with account ID, command type, success flag, elapsed time, and message.

---

## Quick Start

**Requirements:**

- Windows 10 or Windows 11
- .NET 8 SDK
- Visual Studio 2022

```bash
git clone https://github.com/your-username/prop-matrix-engine.git
cd prop-matrix-engine
```

Open `WinFormsApp4.slnx` in Visual Studio and press **F5**.

---

## How to Use

1. Launch the dashboard.
2. Enter an MT5 ID and password.
3. Select the target server profile.
4. Click **Add MT5 Account**.
5. Adjust daily drawdown limit, pre-breach buffer, and copy lot size.
6. Monitor account equity and exposure.
7. Use **GLOBAL KILL SWITCH** when risk must be reduced immediately.

---

## Roadmap

- [x] Multi-account registry
- [x] Global risk evaluation
- [x] Kill-switch command sequence
- [x] Chart-line copy engine
- [x] SignalR client adapter
- [ ] Add persistent account profiles
- [ ] Add broker-specific MT5 bridge adapter
- [ ] Add audit export to CSV
- [ ] Add role-based operator lock

---

## License

MIT

---

<div align="center">

Prop Matrix Engine - Multi-Account MT5 Risk Command Center

</div>
