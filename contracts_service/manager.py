from typing import Dict, Optional
import logging
import os
import json
import time
from datetime import datetime
import requests
from products import Product
from products.lucky_ladder import LuckyLadder
from products.momentum_catcher import MomentumCatcher

logger = logging.getLogger(__name__)

class StorageClient:
    def __init__(self):
        self.base_url = os.getenv('STORAGE_SERVICE_URL', 'http://storage-service:8001')

    def save_contract(self, contract_id: str, product: Product) -> None:
        url = f"{self.base_url}/contract"
        data = {
            "id": contract_id,
            "type": product.__class__.__name__,
            "parameters": {
                "client_id": product.client_id,
                "contract_id": product.contract_id,
                "duration": product.duration,
                "payoff": product.payoff,
                "is_active": product.is_active,
                "start_time": product.start_time,
                "current_price": product.current_price,
                "last_update": product.last_update,
                # Store product-specific parameters
                **({"rungs": product.rungs} if isinstance(product, LuckyLadder) else {}),
                **({"target_movement": product.target_movement} if isinstance(product, MomentumCatcher) else {})
            },
            "created_at": int(time.time() * 1000),
            "is_active": product.is_active,
            "duration": product.duration
        }
        logger.debug(f"Saving contract data: {json.dumps(data, indent=2)}")
        response = requests.post(url, json=data)
        response.raise_for_status()

    def get_contract(self, contract_id: str) -> Optional[dict]:
        url = f"{self.base_url}/contract"
        response = requests.get(url, params={"id": contract_id})
        if response.status_code == 404:
            return None
        response.raise_for_status()
        data = response.json()
        logger.debug(f"Retrieved contract data: {json.dumps(data, indent=2)}")
        return data

    def get_all_contracts(self) -> list:
        url = f"{self.base_url}/contract"
        response = requests.get(url)
        response.raise_for_status()
        data = response.json()
        logger.debug(f"Retrieved all contracts: {json.dumps(data, indent=2)}")
        return data

    def delete_contract(self, contract_id: str) -> None:
        url = f"{self.base_url}/contract"
        response = requests.delete(url, params={"id": contract_id})
        response.raise_for_status()

class ContractManager:
    def __init__(self):
        self.contracts: Dict[str, Product] = {}
        self.storage = StorageClient()
        self._restore_contracts()

    def _create_product_instance(self, contract_type: str, parameters: dict) -> Optional[Product]:
        """Create a new product instance based on type"""
        product_classes = {
            'LuckyLadder': LuckyLadder,
            'MomentumCatcher': MomentumCatcher
        }
        
        product_class = product_classes.get(contract_type)
        if not product_class:
            logger.error(f"Unknown contract type: {contract_type}")
            return None
            
        product = product_class()
        
        # Create init parameters with required fields
        init_params = {
            "client_id": parameters.get("client_id", "system"),
            "contract_id": parameters["contract_id"],
            "duration": parameters["duration"],
            "payoff": parameters["payoff"]
        }
        
        # Add product-specific parameters
        if isinstance(product, LuckyLadder):
            init_params["rungs"] = parameters.get("rungs", [])
        elif isinstance(product, MomentumCatcher):
            init_params["target_movement"] = parameters.get("target_movement", 0)
        
        logger.debug(f"Initializing product with params: {json.dumps(init_params, indent=2)}")
        product.init(init_params)
        
        # Restore additional state if available
        if parameters.get('start_time') is not None:
            product.start_time = parameters['start_time']
            # Only mark as active if the contract hasn't expired
            elapsed_ms = product.get_elapsed_ms()
            was_active = product.is_active
            product.is_active = elapsed_ms < product.duration
            if was_active != product.is_active:
                try:
                    self.storage.save_contract(product.contract_id, product)
                    logger.debug(f"Updated contract active state in storage: {product.contract_id}")
                except Exception as e:
                    logger.error(f"Error saving contract state: {e}")
            logger.debug(f"Restored start_time: {product.start_time}, elapsed_ms: {elapsed_ms}, is_active: {product.is_active}")
        if parameters.get('current_price') is not None:
            product.current_price = parameters['current_price']
        if parameters.get('last_update') is not None:
            product.last_update = parameters['last_update']
        
        # Start the product if it's active and not started
        if product.is_active and not product.start_time:
            logger.debug("Starting product as it's active but not started")
            product.start()
            try:
                self.storage.save_contract(product.contract_id, product)
                logger.debug(f"Saved newly started contract: {product.contract_id}")
            except Exception as e:
                logger.error(f"Error saving started contract: {e}")
        
        return product

    def _restore_contracts(self) -> None:
        """Restore contracts from storage on startup"""
        try:
            stored_contracts = self.storage.get_all_contracts()
            for contract_data in stored_contracts:
                contract_id = contract_data["id"]
                parameters = contract_data["parameters"]
                parameters['contract_id'] = contract_id  # Ensure contract_id is in parameters
                
                logger.debug(f"Restoring contract {contract_id} with data: {json.dumps(contract_data, indent=2)}")
                product = self._create_product_instance(contract_data["type"], parameters)
                if product:
                    # Store all contracts in memory
                    self.contracts[contract_id] = product
                    logger.info(f"Restored contract {contract_id} from storage, is_active: {product.is_active}")
                else:
                    logger.error(f"Failed to restore contract {contract_id}")
        except Exception as e:
            logger.error(f"Error restoring contracts: {e}")

    def add_contract(self, contract_id: str, product: Product) -> None:
        logger.debug(f"Adding contract {contract_id} to manager")
        if contract_id in self.contracts:
            logger.warning(f"Contract {contract_id} already exists, replacing")
        self.contracts[contract_id] = product
        
        try:
            self.storage.save_contract(contract_id, product)
            logger.debug(f"Saved new contract: {contract_id}")
        except Exception as e:
            logger.error(f"Error saving contract to storage: {e}")
        
        logger.debug(f"Contract {contract_id} added with duration: {product.duration}ms, is_active: {product.is_active}")

    def get_product(self, contract_id: str) -> Optional[Product]:
        # First check in-memory cache
        product = self.contracts.get(contract_id)
        if product:
            # Check if contract has expired
            elapsed_ms = product.get_elapsed_ms()
            if elapsed_ms >= product.duration and product.is_active:
                # Contract just expired, update state and save
                was_active = product.is_active
                product.is_active = False
                if was_active != product.is_active:
                    try:
                        self.storage.save_contract(contract_id, product)
                        logger.debug(f"Updated expired contract state in storage: {contract_id}")
                    except Exception as e:
                        logger.error(f"Error saving expired contract state: {e}")
            logger.debug(f"Found contract {contract_id} in manager, is_active: {product.is_active}, duration: {product.duration}ms, elapsed time: {elapsed_ms}ms")
            return product

        # If not in memory, try to retrieve from storage
        logger.debug(f"Contract {contract_id} not found in manager, checking storage")
        stored_contract = self.storage.get_contract(contract_id)
        if stored_contract:
            parameters = stored_contract["parameters"]
            parameters['contract_id'] = contract_id
            product = self._create_product_instance(stored_contract["type"], parameters)
            if product:
                # Store all contracts in memory
                self.contracts[contract_id] = product
                logger.debug(f"Restored contract {contract_id} from storage, is_active: {product.is_active}, duration: {product.duration}ms, elapsed time: {product.get_elapsed_ms()}ms")
                return product
            
        logger.debug(f"Contract {contract_id} not found in storage")
        return None

    def remove_contract(self, contract_id: str) -> None:
        logger.debug(f"Removing contract {contract_id} from manager")
        if contract_id in self.contracts:
            product = self.contracts[contract_id]
            logger.debug(f"Contract {contract_id} final state - is_active: {product.is_active}, duration: {product.duration}ms, elapsed time: {product.get_elapsed_ms()}ms")
            
            try:
                self.storage.delete_contract(contract_id)
            except Exception as e:
                logger.error(f"Error deleting contract from storage: {e}")
        
        self.contracts.pop(contract_id, None)
