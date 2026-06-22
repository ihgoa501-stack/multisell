"""平台 API 限流器 — 简单令牌桶实现。"""

import asyncio
import time
from collections import defaultdict


class TokenBucket:
    """令牌桶限流器"""

    def __init__(self, rate: float, burst: int):
        self.rate = rate
        self.burst = burst
        self.tokens = float(burst)
        self.last_refill = time.monotonic()

    async def acquire(self):
        now = time.monotonic()
        elapsed = now - self.last_refill
        self.tokens = min(float(self.burst), self.tokens + elapsed * self.rate)
        self.last_refill = now
        if self.tokens < 1:
            wait = (1 - self.tokens) / self.rate
            await asyncio.sleep(wait)
            self.tokens = 0
            self.last_refill = time.monotonic()
        else:
            self.tokens -= 1


_lock = asyncio.Lock()
_limiters: dict[str, TokenBucket] = {}


async def get_limiter_for_platform(platform_code: str, platform_id: int) -> TokenBucket:
    """获取平台限流器，按 platform_code 共享。

    ponytail: 全局 in-memory 限流，多进程场景需 Redis。
    """
    key = f"{platform_code}:{platform_id}"
    async with _lock:
        if key not in _limiters:
            _limiters[key] = TokenBucket(rate=2.0, burst=5)
        return _limiters[key]
