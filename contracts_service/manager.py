from typing import Dict, Optional
import logging
from products import Product

logger = logging.getLogger(__name__)

class ContractManager:
    def __init__(self):
        self.contracts: Dict[str, Product] = {}

    def add_contract(self, contract_id: str, product: Product) -> None:
        logger.debug(f"Adding contract {contract_id} to manager")
        if contract_id in self.contracts:
            logger.warning(f"Contract {contract_id} already exists, replacing")
        self.contracts[contract_id] = product
        logger.debug(f"Contract {contract_id} added with duration: {product.duration}ms, is_active: {product.is_active}")

    def get_product(self, contract_id: str) -> Optional[Product]:
        product = self.contracts.get(contract_id)
        if product:
            logger.debug(f"Found contract {contract_id} in manager, is_active: {product.is_active}, duration: {product.duration}ms, elapsed time: {product.get_elapsed_ms()}ms")
        else:
            logger.debug(f"Contract {contract_id} not found in manager")
        return product

    def remove_contract(self, contract_id: str) -> None:
        logger.debug(f"Removing contract {contract_id} from manager")
        if contract_id in self.contracts:
            product = self.contracts[contract_id]
            logger.debug(f"Contract {contract_id} final state - is_active: {product.is_active}, duration: {product.duration}ms, elapsed time: {product.get_elapsed_ms()}ms")
        self.contracts.pop(contract_id, None)
