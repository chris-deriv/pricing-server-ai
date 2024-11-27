from typing import Dict, Any, Optional
import logging
from .base import Product

logger = logging.getLogger(__name__)

class MomentumCatcher(Product):
    def __init__(self):
        super().__init__()
        self.target_movement: float = 0.0
        self.last_price: Optional[float] = None
        self.max_movement: float = 0.0

    def init(self, params: Dict[str, Any]) -> None:
        super().init(params)
        self.target_movement = params["target_movement"]
        logger.debug(f"Initialized MomentumCatcher contract {self.contract_id} with target movement: {self.target_movement}")
    
    def process_price(self, price: float) -> Dict[str, Any]:
        if self.last_price is None:
            self.last_price = price
            return {
                "status": "active",
                "price": price,
                "movement": 0.0,
                "max_movement": 0.0,
                "target_movement": self.target_movement,
                "target_hit": False
            }
        
        movement = abs(price - self.last_price)
        self.max_movement = max(self.max_movement, movement)
        target_hit = self.max_movement >= abs(self.target_movement)
        
        result = {
            "status": "active",
            "price": price,
            "movement": movement,
            "max_movement": self.max_movement,
            "target_movement": self.target_movement,
            "target_hit": target_hit
        }

        self.last_price = price
        
        if target_hit:
            self.is_active = False
            result["status"] = "target_hit"
            
        return result
