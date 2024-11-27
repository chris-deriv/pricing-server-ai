const WebSocket = require('ws');
const axios = require('axios');

describe('Contract WebSocket Tests', () => {
    let ws;
    const PORT = 8080;
    const WS_URL = `ws://localhost:${PORT}/ws`;
    
    // Helper function to create a promise that resolves with the next message
    const getNextMessage = (timeout = 5000) => {
        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                reject(new Error('Message timeout'));
            }, timeout);

            ws.once('message', (data) => {
                clearTimeout(timer);
                resolve(JSON.parse(data.toString()));
            });

            ws.once('error', (error) => {
                clearTimeout(timer);
                reject(error);
            });

            ws.once('close', () => {
                clearTimeout(timer);
                reject(new Error('Connection closed'));
            });
        });
    };

    beforeAll(async () => {
        // Clean database before starting tests
        await axios.post('http://localhost:8001/clean');
    });

    beforeEach(async () => {
        // Create a new WebSocket connection
        ws = new WebSocket(WS_URL);
        await new Promise((resolve, reject) => {
            ws.once('open', resolve);
            ws.once('error', reject);
        });
    });

    afterEach(() => {
        // Close the WebSocket connection after each test
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.close();
        }
    });

    describe('MomentumCatcher Contract Tests', () => {
        const momentumContract = {
            type: "ContractSubmission",
            data: {
                productType: "MomentumCatcher",
                targetMovement: 5.0,
                duration: 2000,
                payoff: 100.0
            }
        };

        test('should accept valid MomentumCatcher contract', async () => {
            ws.send(JSON.stringify(momentumContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });

        test('should receive contract updates for MomentumCatcher', async () => {
            ws.send(JSON.stringify(momentumContract));
            await getNextMessage(); // Skip the ContractAccepted message
            
            const update = await getNextMessage();
            expect(update.type).toBe('ContractUpdate');
            expect(update.data).toHaveProperty('price');
            expect(update.data).toHaveProperty('movement');
            expect(update.data).toHaveProperty('target_movement');
            expect(update.data).toHaveProperty('status');
            expect(update.data.status).toBe('active');
        });

        test('should reject MomentumCatcher with negative target movement', async () => {
            const invalidContract = {
                ...momentumContract,
                data: {
                    ...momentumContract.data,
                    targetMovement: -5.0
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('targetMovement');
            expect(response.message).toContain('must be positive');
        });

        test('should reject MomentumCatcher with zero target movement', async () => {
            const invalidContract = {
                ...momentumContract,
                data: {
                    ...momentumContract.data,
                    targetMovement: 0
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('targetMovement');
            expect(response.message).toContain('must be positive');
        });
    });

    describe('LuckyLadder Contract Tests', () => {
        const ladderContract = {
            type: "ContractSubmission",
            data: {
                productType: "LuckyLadder",
                rungs: [105, 110, 115],
                duration: 5000,
                payoff: 100
            }
        };

        test('should accept valid LuckyLadder contract', async () => {
            ws.send(JSON.stringify(ladderContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });

        test('should receive contract updates for LuckyLadder', async () => {
            ws.send(JSON.stringify(ladderContract));
            await getNextMessage(); // Skip the ContractAccepted message
            
            const update = await getNextMessage();
            expect(update.type).toBe('ContractUpdate');
            expect(update.data).toHaveProperty('price');
            expect(update.data).toHaveProperty('rungs_hit');
            expect(update.data).toHaveProperty('remaining_rungs');
            expect(update.data).toHaveProperty('status');
            expect(update.data.status).toBe('active');
            expect(Array.isArray(update.data.rungs_hit)).toBe(true);
            expect(Array.isArray(update.data.remaining_rungs)).toBe(true);
        });

        test('should reject LuckyLadder with non-ascending rungs', async () => {
            const invalidContract = {
                ...ladderContract,
                data: {
                    ...ladderContract.data,
                    rungs: [115, 110, 105] // Descending rungs
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('rungs');
            expect(response.message).toContain('ascending order');
        });

        test('should reject LuckyLadder with duplicate rungs', async () => {
            const invalidContract = {
                ...ladderContract,
                data: {
                    ...ladderContract.data,
                    rungs: [105, 105, 110] // Duplicate rung
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('rung values');
            expect(response.message).toContain('not allowed');
        });
    });

    describe('Error Handling Tests', () => {
        test('should handle malformed JSON with error response', async () => {
            ws.send('invalid json{');
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ParseError');
            expect(response.message).toContain('Invalid JSON format');
        });

        test('should reject unknown product type', async () => {
            const invalidContract = {
                type: "ContractSubmission",
                data: {
                    productType: "InvalidProduct",
                    duration: 1000,
                    payoff: 100
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('unsupported product type');
        });

        test('should reject contract with missing required fields', async () => {
            const invalidContract = {
                type: "ContractSubmission",
                data: {
                    productType: "MomentumCatcher",
                    // Missing targetMovement
                    duration: 2000,
                    payoff: 100.0
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('targetMovement');
            expect(response.message).toContain('must be positive');
        });

        test('should reject contract with invalid duration', async () => {
            const invalidContract = {
                type: "ContractSubmission",
                data: {
                    productType: "MomentumCatcher",
                    targetMovement: 5.0,
                    duration: -1000, // Negative duration
                    payoff: 100.0
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('duration');
            expect(response.message).toContain('must be positive');
        });

        test('should reject contract with invalid payoff', async () => {
            const invalidContract = {
                type: "ContractSubmission",
                data: {
                    productType: "MomentumCatcher",
                    targetMovement: 5.0,
                    duration: 2000,
                    payoff: -100.0 // Negative payoff
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('payoff');
            expect(response.message).toContain('must be positive');
        });

        test('should reject contract with missing type field', async () => {
            const invalidContract = {
                // Missing type field
                data: {
                    productType: "MomentumCatcher",
                    targetMovement: 5.0,
                    duration: 2000,
                    payoff: 100.0
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('type');
            expect(response.message).toContain('required');
        });

        test('should reject contract with missing data field', async () => {
            const invalidContract = {
                type: "ContractSubmission"
                // Missing data field
            };
            
            ws.send(JSON.stringify(invalidContract));
            const response = await getNextMessage();
            
            expect(response.type).toBe('Error');
            expect(response.errorType).toBe('ValidationError');
            expect(response.message).toContain('Data field');
            expect(response.message).toContain('required');
        });
    });
});
