from fastapi import FastAPI, HTTPException, Request
import logging
import json
import uuid
from datetime import datetime

from models import ContractRequest
from products import LuckyLadder, MomentumCatcher
from manager import ContractManager

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

app = FastAPI()

# Global manager
contract_manager = ContractManager()

@app.get("/contracts/active")
async def get_active_contracts():
    """Get a list of active contract IDs"""
    active_contracts = []
    for contract_id, product in contract_manager.contracts.items():
        if product.is_active:
            active_contracts.append(contract_id)
    logger.debug(f"Found {len(active_contracts)} active contracts")
    return {"contracts": active_contracts}

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
        
        # Ensure duration is an integer
        if "duration" not in params:
            raise HTTPException(status_code=400, detail="duration is required")
        try:
            duration = int(params["duration"])  # milliseconds
            logger.debug(f"Parsed duration: {duration} ms")
        except ValueError:
            raise HTTPException(status_code=400, detail="duration must be an integer")
        
        if contract_type == "lucky_ladder":
            if "rungs" not in params:
                raise HTTPException(status_code=400, detail="Rungs are required for LuckyLadder")
            product = LuckyLadder()
            init_params = {
                "client_id": "system",
                "contract_id": contract_id,
                "rungs": params["rungs"],
                "duration": duration,
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
                "duration": duration,
                "payoff": params["payoff"]
            }
        else:
            raise HTTPException(status_code=400, detail=f"Unsupported product type: {contract_type}")

        logger.debug(f"Initializing product with params: {json.dumps(init_params, indent=2)}")

        product.init(init_params)
        product.start()
        contract_manager.add_contract(contract_id, product)

        logger.debug(f"Contract {contract_id} created and started, is_active: {product.is_active}, duration: {product.duration}ms")

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

        # Parse timestamp from request if provided, otherwise use current time
        timestamp_str = data.get("timestamp")
        if timestamp_str:
            try:
                timestamp = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
                logger.debug(f"Using timestamp from request: {timestamp}")
            except ValueError:
                logger.warning(f"Invalid timestamp format: {timestamp_str}, using current time")
                timestamp = datetime.now()
        else:
            timestamp = datetime.now()
            logger.debug(f"No timestamp in request, using current time: {timestamp}")
        
        # Handle price update - this will update is_active if contract expires
        result = product.handle_price_update(price, timestamp)
        
        # If contract status changed to expired, save the state
        if result.get("status") == "expired":
            try:
                contract_manager.storage.save_contract(contract_id, product)
                logger.debug(f"Updated expired contract state in storage: {contract_id}")
            except Exception as e:
                logger.error(f"Error saving expired contract state: {e}")
        
        result["contractID"] = contract_id
        result["timestamp"] = timestamp.isoformat()
        
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

@app.get("/contracts/{contract_id}/state")
async def get_contract_state(contract_id: str):
    product = contract_manager.get_product(contract_id)
    if not product:
        raise HTTPException(status_code=404, detail="Contract not found")
    
    # Get elapsed time
    elapsed_ms = product.get_elapsed_ms()
    
    # Build state response
    state = {
        "status": "active" if product.is_active else "inactive",
        "elapsed_ms": elapsed_ms,
        "duration": product.duration,
        "price": product.current_price,
        "product_type": product.__class__.__name__  # Add product type to response
    }
    
    # Add product-specific state
    if isinstance(product, LuckyLadder):
        state.update({
            "rungs_hit": product.last_update.get("rungs_hit", []) if product.last_update else [],
            "remaining_rungs": product.last_update.get("remaining_rungs", product.rungs) if product.last_update else product.rungs
        })
    elif isinstance(product, MomentumCatcher):
        state.update({
            "movement": product.last_update.get("movement", 0) if product.last_update else 0,
            "target_movement": product.target_movement
        })
    
    # If contract has expired, update status
    if elapsed_ms >= product.duration:
        state["status"] = "expired"
    
    return state

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
