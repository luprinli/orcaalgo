#!/usr/bin/env python3
"""
Heston SV + Jump Diffusion Generator (STRESS TESTING ONLY)
===========================================================
Generates OHLCV data using Heston Stochastic Volatility
model with optional Merton Jump Diffusion overlay.

WARNING: This tool generates artificial data for STRESS TESTING.
It is NOT a substitute for real market data. Use orca-fetch --source=stooq
to load production data from the Stooq historical database.

Models:
  - Heston SV: dS = mu*S*dt + sqrt(v)*S*dW1, dv = kappa*(theta - v)*dt + sigma*sqrt(v)*dW2
  - Jump Diffusion: Poisson arrivals with log-normal jump sizes

Usage:
  python -m orca.simulation.synthetic --symbol SPY --bars 5000 --output /tmp/test.csv
  python -m orca.simulation.synthetic --symbol SPY --bars 5000 --jumps --json --seed 42

Dependencies:
  pip install numpy pandas
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from pathlib import Path

import numpy as np
import pandas as pd


@dataclass(frozen=True)
class HestonParams:
    mu: float = 0.05
    kappa: float = 2.0
    theta: float = 0.04
    sigma: float = 0.3
    rho: float = -0.7
    v0: float = 0.04

    def validate(self) -> list[str]:
        errors = []
        if self.kappa <= 0:
            errors.append("kappa must be positive")
        if self.theta <= 0:
            errors.append("theta must be positive")
        if self.sigma <= 0:
            errors.append("sigma must be positive")
        if self.v0 <= 0:
            errors.append("v0 must be positive")
        if abs(self.rho) > 1:
            errors.append("rho must be in [-1, 1]")
        if 2 * self.kappa * self.theta <= self.sigma**2:
            errors.append("Feller condition violated (2*kappa*theta <= sigma^2)")
        return errors


@dataclass(frozen=True)
class JumpParams:
    jump_intensity: float = 0.1
    jump_mean: float = 0.0
    jump_std: float = 0.02

    def validate(self) -> list[str]:
        errors = []
        if self.jump_intensity < 0:
            errors.append("jump_intensity must be non-negative")
        if self.jump_std < 0:
            errors.append("jump_std must be non-negative")
        return errors


@dataclass(frozen=True)
class SyntheticConfig:
    symbol: str = "SYNTH"
    bars: int = 5000
    initial_price: float = 100.0
    dt: float = 1.0 / 252
    heston: HestonParams = HestonParams()
    jumps: JumpParams | None = None
    add_bid_ask: bool = True
    spread_pct: float = 0.05
    seed: int = 42


def generate_heston_prices(
    S0: float,
    params: HestonParams,
    dt: float,
    n_steps: int,
    rng: np.random.Generator,
) -> np.ndarray:
    S = np.zeros(n_steps)
    v = np.zeros(n_steps)
    S[0] = S0
    v[0] = params.v0

    for t in range(n_steps - 1):
        sqrt_v = np.sqrt(max(v[t], 0))
        dW1 = rng.normal(0, np.sqrt(dt))
        dW2 = params.rho * dW1 + np.sqrt(1 - params.rho**2) * rng.normal(0, np.sqrt(dt))

        v[t + 1] = (
            v[t] + params.kappa * (params.theta - max(v[t], 0)) * dt + params.sigma * sqrt_v * dW2
        )
        v[t + 1] = max(v[t + 1], 0)

        S[t + 1] = S[t] * np.exp((params.mu - 0.5 * max(v[t], 0)) * dt + sqrt_v * dW1)

    return S


def apply_jump_diffusion(
    prices: np.ndarray,
    params: JumpParams,
    rng: np.random.Generator,
    dt: float = 1.0 / 252,
) -> np.ndarray:
    result = prices.copy()

    for t in range(len(prices)):
        n_jumps = rng.poisson(params.jump_intensity * dt)
        if n_jumps > 0:
            for _ in range(n_jumps):
                jump_size = rng.normal(params.jump_mean, params.jump_std)
                result[t] *= np.exp(jump_size)

    return result


def prices_to_ohlcv(
    prices: np.ndarray,
    symbol: str,
    freq_minutes: int = 5,
    add_spread: bool = False,
    spread_pct: float = 0.05,
    rng: np.random.Generator | None = None,
) -> pd.DataFrame:
    n = len(prices)
    group_size = max(1, int(freq_minutes / (6.5 * 60 * (1.0 / 252))))
    if group_size < 1:
        group_size = 1

    records = []

    for i in range(0, n, group_size):
        group = prices[i : min(i + group_size, n)]
        if len(group) == 0:
            continue

        open_price = group[0]
        close_price = group[-1]
        high_price = max(group)
        low_price = min(group)

        if add_spread and rng is not None:
            half_spread = spread_pct / 200.0
            noise = rng.uniform(-half_spread, half_spread)
            open_price *= 1 + noise
            close_price *= 1 + noise
            high_price *= 1 + half_spread
            low_price *= 1 - half_spread

        volume = rng.exponential(100000) + 1000 if rng else 100000

        records.append(
            {
                "symbol": symbol,
                "open": round(open_price, 4),
                "high": round(high_price, 4),
                "low": round(low_price, 4),
                "close": round(close_price, 4),
                "volume": int(volume),
            }
        )

    return pd.DataFrame(records)


def generate_synthetic_data(config: SyntheticConfig) -> pd.DataFrame:
    rng = np.random.default_rng(config.seed)

    prices = generate_heston_prices(
        S0=config.initial_price,
        params=config.heston,
        dt=config.dt,
        n_steps=config.bars,
        rng=rng,
    )

    if config.jumps is not None:
        prices = apply_jump_diffusion(prices, config.jumps, rng, dt=config.dt)

    df = prices_to_ohlcv(
        prices=prices,
        symbol=config.symbol,
        add_spread=config.add_bid_ask,
        spread_pct=config.spread_pct,
        rng=rng,
    )

    return df


def main() -> None:
    parser = argparse.ArgumentParser(description="Synthetic market data generator")
    parser.add_argument("--symbol", type=str, default="SYNTH", help="Ticker symbol")
    parser.add_argument("--bars", type=int, default=5000, help="Number of price bars to generate")
    parser.add_argument("--price", type=float, default=100.0, help="Initial price")
    parser.add_argument("--mu", type=float, default=0.05, help="Drift (annualized)")
    parser.add_argument("--kappa", type=float, default=2.0, help="Mean reversion speed")
    parser.add_argument("--theta", type=float, default=0.04, help="Long-term variance")
    parser.add_argument("--sigma", type=float, default=0.3, help="Vol-of-vol")
    parser.add_argument("--rho", type=float, default=-0.7, help="Correlation")
    parser.add_argument("--jumps", action="store_true", help="Enable jump diffusion overlay")
    parser.add_argument("--jump-intensity", type=float, default=0.1, help="Poisson jump intensity")
    parser.add_argument("--jump-mean", type=float, default=0.0, help="Jump mean")
    parser.add_argument("--jump-std", type=float, default=0.02, help="Jump std dev")
    parser.add_argument("--no-spread", action="store_true", help="Disable bid-ask spread")
    parser.add_argument("--spread-pct", type=float, default=0.05, help="Bid-ask spread %")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    parser.add_argument("--output", type=str, default=None, help="Output CSV path")
    parser.add_argument("--json", action="store_true", help="Output JSON to stdout")
    args = parser.parse_args()

    heston = HestonParams(
        mu=args.mu / 100.0 if args.mu > 1 else args.mu,
        kappa=args.kappa,
        theta=args.theta / 100.0 if args.theta > 1 else args.theta,
        sigma=args.sigma,
        rho=args.rho,
        v0=args.theta / 100.0 if args.theta > 1 else args.theta,
    )

    errors = heston.validate()
    if errors:
        for e in errors:
            print(f"WARNING: {e}", file=sys.stderr)

    jump_params = None
    if args.jumps:
        jump_params = JumpParams(
            jump_intensity=args.jump_intensity,
            jump_mean=args.jump_mean,
            jump_std=args.jump_std,
        )

    config = SyntheticConfig(
        symbol=args.symbol,
        bars=args.bars,
        initial_price=args.price,
        heston=heston,
        jumps=jump_params,
        add_bid_ask=not args.no_spread,
        spread_pct=args.spread_pct,
        seed=args.seed,
    )

    df = generate_synthetic_data(config)

    if args.json:
        output = []
        for _, row in df.iterrows():
            output.append(
                {
                    "symbol": row["symbol"],
                    "open": float(row["open"]),
                    "high": float(row["high"]),
                    "low": float(row["low"]),
                    "close": float(row["close"]),
                    "volume": int(row["volume"]),
                }
            )
        print(json.dumps(output))
        return

    if args.output:
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        df.to_csv(output_path, index=False)
        print(f"Generated {len(df)} bars -> {output_path}")
        return

    print(df.head(10).to_string())
    print(f"\nGenerated {len(df)} OHLCV bars for {args.symbol}")
    print(f"Price range: ${df['close'].min():.2f} - ${df['close'].max():.2f}")
    print(f"Annualized vol: {df['close'].pct_change().std() * np.sqrt(252) * 100:.1f}%")


if __name__ == "__main__":
    main()
