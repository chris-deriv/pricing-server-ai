from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel
from typing import Dict, Any, Literal, List, Optional
from abc import ABC, abstractmethod
from datetime import datetime, timedelta
import logging
import json
import uuid

app = FastAPI()

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class PriceUpdate(BaseModel):
    price: float

class ContractParameters(BaseModel):
    contract_id: Optional[str] = None
    duration: int = 300  # duration in seconds
    payoff: float = 0.0
    rungs: Optional[List[float]] = None
    target_movement: Optional[float] = None

class ContractRequest(BaseModel):
    contract_type: Literal["lucky_ladder", "momentum_catcher"]
    parameters: ContractParameters

class Product(ABC):
    def __init__(self):
        self.client_id: str = ""
        self.contract_id: str = ""
        self.duration: int = 300
        self.payoff: float = 0.0
        self.start_time: Optional[datetime] = None
        self.is_active: bool = False  # Will be set to True in start()
        self.last_update: Optional[Dict[str, Any]] = None
        self.current_price: Optional[float] = None

    @abstractmethod
    def init(self, params: Dict[str, Any]) -> None:
        logger.debug(f"Initializing contract {params['contract_id']}")
        self.client_id = params["client_id"]
        self.contract_id = params["contract_id"]
        self.duration = params["duration"]
        self.payoff = params["payoff"]
        logger.debug(f"Contract {self.contract_id} initialized with duration: {self.duration} seconds")

    def start(self) -> None:
        logger.debug(f"Starting contract {self.contract_id}")
        self.start_time = datetime.now()
        self.is_active = True
        logger.debug(f"Contract {self.contract_id} started at {self.start_time}, is_active: {self.is_active}, duration: {self.duration} seconds")

    def handle_price_update(self, price: float, timestamp: datetime) -> Dict[str, Any]:
        logger.debug(f"Handling price update for contract {self.contract_id}")
        logger.debug(f"Contract state - is_active: {self.is_active}, start_time: {self.start_time}, duration: {self.duration} seconds")
        
        self.current_price = price
        
        # Check if contract has been started
        if self.start_time is None:
            logger.debug(f"Contract {self.contract_id} hasn't been started yet")
            self.start()
        
        if not self.is_active:
            logger.debug(f"Contract {self.contract_id} is inactive")
            return {"status": "inactive", "price": price}
        
        # Check expiry
        time_elapsed = timestamp - self.start_time
        logger.debug(f"Contract {self.contract_id} time elapsed: {time_elapsed.total_seconds()}s, duration: {self.duration}s")
        
        if time_elapsed >= timedelta(seconds=self.duration):
            logger.debug(f"Contract {self.contract_id} expired (elapsed: {time_elapsed.total_seconds()}s >= duration: {self.duration}s)")
            self.is_active = False
            return {"status": "expired", "price": price}
            
        result = self.process_price(price)
        self.last_update = result
        logger.debug(f"Contract {self.contract_id} processed price: {result}")
        return result
    
    @abstractmethod
    def process_price(self, price: float) -> Dict[str, Any]:
        pass

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

class ContractManager:
    def __init__(self):
        self.contracts: Dict[str, Product] = {}

    def add_contract(self, contract_id: str, product: Product) -> None:
        logger.debug(f"Adding contract {contract_id} to manager")
        self.contracts[contract_id] = product

    def get_product(self, contract_id: str) -> Optional[Product]:
        product = self.contracts.get(contract_id)
        if product:
            logger.debug(f"Found contract {contract_id} in manager, is_active: {product.is_active}, duration: {product.duration}s")
        else:
            logger.debug(f"Contract {contract_id} not found in manager")
        return product

    def remove_contract(self, contract_id: str) -> None:
        logger.debug(f"Removing contract {contract_id} from manager")
        self.contracts.pop(contract_id, None)

# Global manager
contract_manager = ContractManager()

@app.post("/contracts")
async def create_contract(request: Request):
    try:
        body = await request.body()
        logger.debug(f"Received raw request body: {body.decode()}")
        
        data = await request.json()
        logger.debug(f"Parsed request data: {json.dumps(data, indent=2)}")
        
        if "contract_type" not in data:
            raise HTTPException(status_code=400, detail="contract_type is required")
        if "parameters" not in data:
            raise HTTPException(status_code=400, detail="parameters is required")

        contract_type = data["contract_type"]
        params = data["parameters"]
        contract_id = params.get("contract_id") or str(uuid.uuid4())
        
        if contract_type == "lucky_ladder":
            if "rungs" not in params:
                raise HTTPException(status_code=400, detail="Rungs are required for LuckyLadder")
            product = LuckyLadder()
            init_params = {
                "client_id": "system",
                "contract_id": contract_id,
                "rungs": params["rungs"],
                "duration": params["duration"],
                "payoff": params["payoff"]
            }
        elif contract_type == "momentum_catcher":
            if "target_movement" not in params:
                raise HTTPException(status_code=400, detail="Target movement is required for MomentumCatcher")
            product = MomentumCatcher()
            init_params = {
                "client_id": "system",
                "contract_id": contract_id,
                "target_movement": params["target_movement"],
                "duration": params["duration"],
                "payoff": params["payoff"]
            }
        else:
            raise HTTPException(status_code=400, detail=f"Unsupported product type: {contract_type}")

        logger.debug(f"Initializing product with params: {json.dumps(init_params, indent=2)}")

        product.init(init_params)
        product.start()
        contract_manager.add_contract(contract_id, product)

        logger.debug(f"Contract {contract_id} created and started, is_active: {product.is_active}, duration: {product.duration}s")

        return {
            "status": "success",
            "contract_id": contract_id
        }

    except json.JSONDecodeError as e:
        logger.error(f"JSON decode error: {str(e)}")
        raise HTTPException(status_code=400, detail=f"Invalid JSON: {str(e)}")
    except Exception as e:
        logger.error(f"Error processing contract request: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

@app.post("/contracts/{contract_id}/price-update")
async def update_price(contract_id: str, request: Request):
    try:
        body = await request.body()
        logger.debug(f"Received price update for {contract_id}: {body.decode()}")
        
        data = await request.json()
        logger.debug(f"Parsed price update data: {json.dumps(data, indent=2)}")
        
        product = contract_manager.get_product(contract_id)
        if not product:
            raise HTTPException(status_code=404, detail="Contract not found")
        
        price = data.get("price")
        if price is None:
            raise HTTPException(status_code=400, detail="Price is required")
        
        result = product.handle_price_update(price, datetime.now())
        result["contractID"] = contract_id
        result["timestamp"] = datetime.now().isoformat()
        
        logger.debug(f"Price update result: {json.dumps(result, indent=2)}")
        return result
        
    except json.JSONDecodeError as e:
        logger.error(f"JSON decode error: {str(e)}")
        raise HTTPException(status_code=400, detail=f"Invalid JSON: {str(e)}")
    except Exception as e:
        logger.error(f"Error processing price update: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

@app.get("/contracts/{contract_id}/price-update")
async def get_last_update(contract_id: str):
    product = contract_manager.get_product(contract_id)
    if not product:
        raise HTTPException(status_code=404, detail="Contract not found")
    
    if product.last_update is None:
        return {
            "status": "active" if product.is_active else "inactive",
            "contractID": contract_id,
            "price": product.current_price,
            "timestamp": datetime.now().isoformat()
        }
    
    # Ensure contractID and timestamp are in the response
    product.last_update["contractID"] = contract_id
    product.last_update["timestamp"] = datetime.now().isoformat()
    return product.last_update

@app.delete("/contracts/{contract_id}")
async def remove_contract(contract_id: str):
    if contract_manager.get_product(contract_id) is None:
        raise HTTPException(status_code=404, detail="Contract not found")
    
    contract_manager.remove_contract(contract_id)
    return {"status": "success"}

@app.get("/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
