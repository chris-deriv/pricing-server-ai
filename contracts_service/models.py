from pydantic import BaseModel
from typing import List, Optional, Literal

class PriceUpdate(BaseModel):
    price: float

class ContractParameters(BaseModel):
    contract_id: Optional[str] = None
    duration: int = 300000  # duration in milliseconds (default 5 minutes)
    payoff: float = 0.0
    rungs: Optional[List[float]] = None
    target_movement: Optional[float] = None

class ContractRequest(BaseModel):
    contract_type: Literal["lucky_ladder", "momentum_catcher"]
    parameters: ContractParameters
