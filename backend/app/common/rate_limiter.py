"""Platform API rate limiters — token-bucket per platform.

Default rates (requests/second, burst):
  ozon:    3/10   — Ozon Seller API doc recommends ≤3 req/s
  shopee:  5/15   — Shopee Open Platform: ≤30 req/min → 0.5/s, generous burst
  wb:      5/10   — Wildberries: ≤10 req/s
  default: 10/20  — fallback

ponytail: per-account limits if one SB account uses multiple tenants.
"""
from __future__ import annotations

import asyncio
import time
from typing import Any

_RATE_CONFIG: dict[str, tuple[float, int]] = {
    "ozon": (3, 10),
    "shopee": (0.5, 15),
    "wb": (5, 10),
    "default": (10, 20),
}


class _TokenBucket:
    """Async token bucket rate limiter."""

    def __init__(self, rate: float, burst: int) -> None:
        self._rate = rate
        self._capacity = burst
        self._tokens = float(burst)
        self._refilled_at = time.monotonic()
        self._lock = asyncio.Lock()

    async def acquire(self) -> None:
        while True:
            async with self._lock:
                self._refill()
                if self._tokens >= 1:
                    self._tokens -= 1
                    return
                wait = (1 - self._tokens) / self._rate if self._rate > 0 else 1
            await asyncio.sleep(wait)

    def _refill(self) -> None:
        now = time.monotonic()
        elapsed = now - self._refilled_at
        self._tokens = min(self._capacity, self._tokens + elapsed * self._rate)
        self._refilled_at = now


_limiters: dict[str, _TokenBucket] = {}
_init_lock = asyncio.Lock()


async def get_limiter_for_platform(platform_code: str, platform_id: Any = None) -> _TokenBucket:
    async with _init_lock:
        limiter = _limiters.get(platform_code)
        if limiter is None:
            rate, burst = _RATE_CONFIG.get(platform_code, _RATE_CONFIG["default"])
            limiter = _TokenBucket(rate, burst)
            _limiters[platform_code] = limiter
        return limiter
