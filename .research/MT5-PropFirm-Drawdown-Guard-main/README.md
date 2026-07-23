<div align="center">

Topics: metatrader5, mql5, prop-firm, expert-advisor, mql4, metatrader, forex-trading, automated-trading, risk-management, money-management, ftmo, funded-account, drawdown-protection, mt4, mt5, trading-bot, mt5-drawdown-guard, ftmo-equity-protector, prop-firm-kill-switch

# Information

**Drawdown protection and equity guard suite for prop firm traders using MT5 / MT4. The project enforces daily loss limits, total drawdown rules, and equity shields with a hard kill-switch, helping traders survive funded challenges and live phases without breaking the firm's risk parameters.**

# 🛡️ PropFirm Drawdown Guard MT5/MT4

**Real-time equity guard with daily / overall drawdown enforcement, hard kill-switch, and full prop firm rule simulation.**

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
  <img src="https://i.ibb.co/n83kMLBG/6.png" alt="Drawdown guard UI" width="820">
</p>

</div>

---

## 🎬 Demo

<div align="center">

<img src="https://i.imgur.com/Exf7NOq.gif" alt="Demo">

</div>


---

## Why a Drawdown Guard?

Most prop firm failures are not from bad strategies, they are from one bad day that breaks a hard limit.

This project keeps you funded by:
- Watching equity tick by tick  
- Closing trades before the rule breaks  
- Locking the account for the rest of the session  

---

## What It Does

**PropFirm Drawdown Guard MT5/MT4** mirrors typical prop firm risk rules and enforces them locally before the broker server flags you.

| Module | Description |
|---|---|
| Equity Monitor | Tick-level equity tracking |
| Daily Loss Engine | Resets at broker server midnight |
| Max DD Engine | Trailing or static drawdown |
| Kill Switch | Closes all & blocks new orders |
| Buffer Layer | Configurable safety buffer |
| Session Log | Full audit trail of triggers |

---

## Features

| Feature | Description |
|---|---|
| Daily Limit Rule | Loss-of-day shutdown |
| Overall Limit Rule | Max account drawdown |
| Trailing Drawdown | High-water-mark mode |
| Soft Warn / Hard Stop | Two-level alert system |
| MT4 / MT5 Support | Platform selection system |
| Broker Time Sync | Resets exactly on broker midnight |
| Audio Alerts | Pre-trigger warning |
| Account Selector | Multiple challenges side by side |
| Rule Presets | FTMO, MFF, FundedNext, custom |
| Recovery Mode | Manual unlock after review |

---

## System Behavior

- Tick-frequency equity sampling
- Server-time aware, no local-clock drift
- Closes positions sequentially to limit slippage
- Refuses new orders while locked, even from external EAs

---

## Quick Start

**Requirements:**
- Windows 10 / 11  
- .NET 8+  
- Visual Studio 2022  

```bash
git clone https://github.com/your-username/propfirm-drawdown.git
```

Open solution → Press **F5**

---

## How to Use

1. Launch app  
2. Select MT4 / MT5  
3. Enter login  
4. Click **CONNECT**  
5. Load prop firm preset  
6. Set buffer percent  
7. Click **ARM GUARD**  
8. Trade with a hard safety net  

---

## Interface Logic

```
START   ████████████████████   100%
NOW     ██████████████░░░░░░    72%
LIMIT   ████░░░░░░░░░░░░░░░░    25%  <- KILL LINE
```

- Green band = safe zone  
- Yellow band = warning zone  
- Red band = trigger zone  
- Locked panel = guard fired  

---

## Roadmap

- [x] Equity monitor  
- [x] Kill switch  
- [x] Rule presets  
- [ ] Real MT5 EA bridge  
- [ ] Cloud rule sync  
- [ ] Per-symbol risk caps  
- [ ] Mobile alert push  

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

PropFirm Drawdown Guard MT5/MT4 · v1.0

</div>
