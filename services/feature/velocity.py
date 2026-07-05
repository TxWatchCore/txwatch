import time
import redis.asyncio as aioredis
from typing import Optional

VELOCITY_WINDOWS = {
    "1h": 3600,
    "6h": 21600,
    "24h": 86400,
}


class VelocityStore:
    """
    Sliding window velocity counters using Redis sorted sets.
    
    Each user has sorted sets keyed by:
      velocity:{user_id}:count:{window}   — transaction counts
      velocity:{user_id}:amount:{window}  — transaction amounts
    
    Scores are Unix timestamps. Members are transaction IDs.
    """

    def __init__(self, redis_client: aioredis.Redis):
        self.redis = redis_client

    async def record_transaction(
        self,
        user_id: str,
        transaction_id: str,
        amount: float,
        timestamp: Optional[float] = None,
    ):
        """Record a transaction in all velocity windows."""
        ts = timestamp or time.time()
        pipe = self.redis.pipeline()

        for window_name, window_seconds in VELOCITY_WINDOWS.items():
            count_key = f"velocity:{user_id}:count:{window_name}"
            amount_key = f"velocity:{user_id}:amount:{window_name}"
            cutoff = ts - window_seconds

            # Add to count window
            pipe.zadd(count_key, {transaction_id: ts})
            pipe.zremrangebyscore(count_key, 0, cutoff)
            pipe.expire(count_key, window_seconds * 2)

            # Add to amount window
            pipe.zadd(amount_key, {f"{transaction_id}:{amount}": ts})
            pipe.zremrangebyscore(amount_key, 0, cutoff)
            pipe.expire(amount_key, window_seconds * 2)

        await pipe.execute()

    async def get_velocity_features(
        self,
        user_id: str,
        timestamp: Optional[float] = None,
    ) -> dict:
        """
        Returns velocity features for a user at a given timestamp.
        Falls back to zeros if Redis is unavailable.
        """
        ts = timestamp or time.time()
        features = {}

        try:
            pipe = self.redis.pipeline()

            for window_name, window_seconds in VELOCITY_WINDOWS.items():
                count_key = f"velocity:{user_id}:count:{window_name}"
                amount_key = f"velocity:{user_id}:amount:{window_name}"
                cutoff = ts - window_seconds

                pipe.zcount(count_key, cutoff, ts)
                pipe.zrangebyscore(amount_key, cutoff, ts)

            results = await pipe.execute()

            # Parse results — alternating count, amount_members
            idx = 0
            for window_name in VELOCITY_WINDOWS:
                count = results[idx]
                amount_members = results[idx + 1]
                idx += 2

                # Sum amounts from member strings "txn_id:amount"
                total_amount = 0.0
                for member in amount_members:
                    try:
                        amount_str = member.decode() if isinstance(member, bytes) else member
                        total_amount += float(amount_str.split(":")[-1])
                    except (ValueError, IndexError):
                        continue

                features[f"velocity_count_{window_name}"] = int(count)
                features[f"velocity_amount_{window_name}"] = round(total_amount, 2)

        except Exception as e:
            print(f"[velocity] Redis unavailable, using zeros: {e}")
            # Graceful degradation — NFR3
            for window_name in VELOCITY_WINDOWS:
                features[f"velocity_count_{window_name}"] = 0
                features[f"velocity_amount_{window_name}"] = 0.0

        return features
