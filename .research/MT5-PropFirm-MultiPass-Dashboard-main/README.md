<div align="center">

Topics: metatrader5, mql5, prop-firm, expert-advisor, mql4, metatrader, forex-trading, trading-dashboard, risk-management, data-visualization, ftmo, funded-account, prop-firm-challenge, mt4, mt5, trading-bot, mt5-challenge-tracker, ftmo-phase-dashboard, funded-account-tracker

# Information

**Multi-phase prop firm challenge manager and progress tracker for MT5 / MT4 traders. The project handles parallel challenges across phases, profit targets, daily limits, and verification stages, giving funded-account hopefuls a single dashboard to monitor every active pass attempt.**

# 🎯 PropFirm MultiPass MT5/MT4

**Track multiple prop firm challenges in parallel, with phase progress, profit targets, daily caps, and payout countdowns in one dashboard.**

<br>

[![Stars](https://img.shields.io/github/stars/torvalds/linux?style=for-the-badge&color=00D4AA&label=Stars)](https://github.com/your-username/volume-profile-mt5/stargazers)
[![Forks](https://img.shields.io/github/forks/torvalds/linux?style=for-the-badge&color=4D9FFF&label=Forks)](https://github.com/your-username/volume-profile-mt5/network)
[![Issues](https://img.shields.io/github/issues/torvalds/linux?style=for-the-badge&color=FF4D6A&label=Issues)](https://github.com/your-username/volume-profile-mt5/issues)
[![Platform](https://img.shields.io/badge/MT5%20%2F%20MT4-Compatible-00D4AA?style=for-the-badge)](https://www.metatrader5.com)
[![License](https://img.shields.io/badge/License-MIT-4D9FFF?style=for-the-badge)](LICENSE)

</div>

<p align="center">
    <img src="https://minkxx-spotify-readme.vercel.app/api?theme=dark&rainbow=true&scan=true&spin=True" alt="Preview">
</p>

---

## 📸 Screenshot

<div align="center">

<p align="center">
  <img src="https://i.ibb.co/tPpwgsns/7.png" alt="MultiPass dashboard" width="820">
</p>

</div>

---

## 🎬 Demo

<div align="center">

<img src="https://i.imgur.com/7jyBC5D.gif" alt="Demo">

</div>


---

## Why MultiPass?

Serious prop traders rarely run one challenge, they run five.

This project gives you one screen for:
- Phase 1 / Phase 2 / Funded accounts side by side  
- Profit target distance per challenge  
- Days traded, days left, payout windows  

---

## What It Does

**PropFirm MultiPass MT5/MT4** centralizes every active prop firm attempt and projects its real status in numbers and color.

| Module | Description |
|---|---|
| Account Registry | Stores all challenge logins |
| Phase Engine | Phase 1 / 2 / Funded states |
| Target Tracker | Profit % to next pass |
| Day Counter | Min days traded, days left |
| Limit Watcher | Daily and max drawdown alerts |
| Payout Planner | Next payout date and amount |

---

## Features

| Feature | Description |
|---|---|
| Account Grid | All challenges in one table |
| Phase Badges | Phase 1, Phase 2, Funded |
| Target Bars | Profit progress to target |
| Limit Bars | Distance to daily / max DD |
| Days Tracker | Min trading days enforced |
| MT4 / MT5 Support | Platform selection system |
| Firm Presets | FTMO, MFF, FundedNext, custom |
| Payout Countdown | Live timer to next payout |
| P&L Sync | Auto-pulls broker equity |
| Notes Field | Per-challenge journaling |

---

## System Behavior

- Per-account isolated state
- Resilient to broker disconnects
- Color-graded urgency (green to red)
- Persistent storage across app restarts

---

## Quick Start

**Requirements:**
- Windows 10 / 11  
- .NET 8+  
- Visual Studio 2022  

```bash
git clone https://github.com/your-username/propfirm-multipass.git
```

Open solution → Press **F5**

---

## How to Use

1. Launch app  
2. Select MT4 / MT5  
3. Add each challenge account  
4. Pick firm preset per account  
5. Set start date & phase  
6. Click **SYNC ALL**  
7. Review progress grid  
8. Plan the next trading day  

---

## Interface Logic

```
ACC #1  P1  TARGET 8/8  ████████  ✓ READY
ACC #2  P2  TARGET 3/5  ██████░░   60%
ACC #3  FUN PAYOUT 4d  ███████░   PAYDAY
ACC #4  P1  DD WARN     ██░░░░░░  -7.4%
```

- Green row = on track  
- Yellow row = caution  
- Red row = limit risk  
- Star = phase passed  

---

## Roadmap

- [x] Account grid  
- [x] Phase engine  
- [x] Payout tracker  
- [ ] Real MT5 multi-account bridge  
- [ ] Firm rule auto-updates  
- [ ] Mobile sync  
- [ ] Tax / payout reports  

---

## Contributing

```
1. Fork
2. git checkout -b feature/new-feature
3. git commit -m "Add feature"
4. git push
5. Open PR
```

---

## License

MIT

---

<div align="center">

PropFirm MultiPass MT5/MT4 · v1.0

</div>
