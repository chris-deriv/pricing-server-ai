from typing import Dict, Any, List
import logging
from .base import Product

logger = logging.getLogger(__name__)

class LuckyLadder(Product):
    def __init__(self):
        super().__init__()
        self.rungs: List[float] = []
        self.hit_rungs: List[float] = []

    def init(self, params: Dict[str, Any]) -> None:
        super().init(params)
        self.rungs = sorted(params["rungs"])
        logger.debug(f"Initialized LuckyLadder contract {self.contract_id} with rungs: {self.rungs}")
    
    def process_price(self, price: float) -> Dict[str, Any]:
        current_hits = [rung for rung in self.rungs if abs(price - rung) < 0.0001]
        self.hit_rungs.extend(current_hits)
        self.hit_rungs = sorted(list(set(self.hit_rungs)))  # Remove duplicates and sort

        return {
            "status": "active",
            "price": price,
            "rungs_hit": current_hits,
            "all_rungs_hit": self.hit_rungs,
            "remaining_rungs": [r for r in self.rungs if r not in self.hit_rungs]
        }
