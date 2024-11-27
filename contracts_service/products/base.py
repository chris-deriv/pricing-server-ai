from abc import ABC, abstractmethod
from typing import Dict, Any, Optional
from datetime import datetime
import time
import logging

logger = logging.getLogger(__name__)

class Product(ABC):
    def __init__(self):
        self.client_id: str = ""
        self.contract_id: str = ""
        self.duration: int = 300000  # milliseconds
        self.payoff: float = 0.0
        self.start_time: Optional[float] = None  # monotonic time
        self.is_active: bool = False  # Will be set to True in start()
        self.last_update: Optional[Dict[str, Any]] = None
        self.current_price: Optional[float] = None

    @abstractmethod
    def init(self, params: Dict[str, Any]) -> None:
        logger.debug(f"Initializing contract {params['contract_id']}")
        self.client_id = params["client_id"]
        self.contract_id = params["contract_id"]
        self.duration = int(params["duration"])  # milliseconds
        self.payoff = params["payoff"]
        logger.debug(f"Contract {self.contract_id} initialized with duration: {self.duration} ms")

    def start(self) -> None:
        logger.debug(f"Starting contract {self.contract_id}")
        self.start_time = time.monotonic()
        self.is_active = True
        logger.debug(f"Contract {self.contract_id} started at monotonic time {self.start_time}, is_active: {self.is_active}, duration: {self.duration} ms")

    def get_elapsed_ms(self) -> int:
        """Get elapsed time in milliseconds since contract start"""
        if self.start_time is None:
            return 0
        current_time = time.monotonic()
        return int((current_time - self.start_time) * 1000)

    def handle_price_update(self, price: float, timestamp: datetime) -> Dict[str, Any]:
        logger.debug(f"Handling price update for contract {self.contract_id}")
        logger.debug(f"Contract state - is_active: {self.is_active}, start_time: {self.start_time}, duration: {self.duration} ms")
        
        self.current_price = price
        
        # Check if contract has been started
        if self.start_time is None:
            logger.debug(f"Contract {self.contract_id} hasn't been started yet")
            self.start()
            return {
                "status": "active",
                "price": price,
                "elapsed_ms": 0,
                "duration": self.duration
            }
        
        if not self.is_active:
            logger.debug(f"Contract {self.contract_id} is inactive")
            return {"status": "inactive", "price": price}
        
        # Check expiry using monotonic time
        elapsed_ms = self.get_elapsed_ms()
        logger.debug(f"Contract {self.contract_id} time elapsed: {elapsed_ms}ms, duration: {self.duration}ms")
        
        if elapsed_ms >= self.duration:
            logger.debug(f"Contract {self.contract_id} expired (elapsed: {elapsed_ms}ms >= duration: {self.duration}ms)")
            self.is_active = False
            return {
                "status": "expired",
                "price": price,
                "elapsed_ms": elapsed_ms,
                "duration": self.duration
            }
            
        result = self.process_price(price)
        self.last_update = result
        # Add duration info to result for debugging
        result.update({
            "elapsed_ms": elapsed_ms,
            "duration": self.duration
        })
        logger.debug(f"Contract {self.contract_id} processed price: {result}")
        return result
    
    @abstractmethod
    def process_price(self, price: float) -> Dict[str, Any]:
        pass
