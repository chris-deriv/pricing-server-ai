# Contract Data Storage Flow

## 1. Initial Contract Creation
When a contract is created, it starts in `contracts_service/manager.py`:

```python
def add_contract(self, contract_id: str, product: Product) -> None:
    # First stored in memory
    self.contracts[contract_id] = product
    
    # Then persisted to storage service
    try:
        self.storage.save_contract(contract_id, product)
    except Exception as e:
        logger.error(f"Error saving contract to storage: {e}")
```

## 2. Storage Service Call
The StorageClient in manager.py makes an HTTP call to store the data:

```python
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
            "last_update": product.last_update
        },
        "created_at": int(time.time() * 1000),
        "is_active": product.is_active,
        "duration": product.duration
    }
    response = requests.post(url, json=data)
```

## 3. PostgreSQL Storage
The storage service (storage_service/main.go) then stores this in PostgreSQL:

```go
func (s *PostgresStorage) Save(id string, contract *Contract) error {
    _, err := s.db.Exec(`
        INSERT INTO contracts (id, type, parameters, created_at, is_active, duration)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            type = EXCLUDED.type,
            parameters = EXCLUDED.parameters,
            created_at = EXCLUDED.created_at,
            is_active = EXCLUDED.is_active,
            duration = EXCLUDED.duration
    `, contract.ID, contract.Type, contract.Parameters, contract.CreatedAt, 
       contract.IsActive, contract.Duration)
    return err
}
```

## 4. Data Recovery
When the service restarts, ContractManager restores contracts from storage:

```python
def _restore_contracts(self) -> None:
    try:
        stored_contracts = self.storage.get_all_contracts()
        for contract_data in stored_contracts:
            if contract_data["is_active"]:
                contract_id = contract_data["id"]
                parameters = contract_data["parameters"]
                parameters['contract_id'] = contract_id
                
                product = self._create_product_instance(contract_data["type"], parameters)
                if product:
                    self.contracts[contract_id] = product
    except Exception as e:
        logger.error(f"Error restoring contracts: {e}")
```

## Storage Location
The actual contract data is stored in:

1. **Runtime Memory**: In the ContractManager's self.contracts dictionary
2. **Persistent Storage**: In PostgreSQL database running in the 'db' container
   - Database: contracts
   - Table: contracts
   - Data Location: /var/lib/postgresql/data (inside the container)
   - Volume Mount: postgres_data (Docker volume)

The PostgreSQL data is persisted through the Docker volume 'postgres_data', ensuring it survives container restarts.
