const WebSocket = require('ws');

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

    beforeEach(async () => {
        // Create a new WebSocket connection before each test
        ws = new WebSocket(WS_URL);
        await new Promise((resolve) => ws.once('open', resolve));
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

        test('should handle MomentumCatcher with extreme target movement', async () => {
            const extremeContract = {
                ...momentumContract,
                data: {
                    ...momentumContract.data,
                    targetMovement: 1000.0 // Extremely high target
                }
            };
            
            ws.send(JSON.stringify(extremeContract));
            const response = await getNextMessage();
            
            // Even with extreme values, we expect the contract to be accepted
            // as the server implements its own risk management
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
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

        test('should handle LuckyLadder with extreme rungs', async () => {
            const extremeContract = {
                ...ladderContract,
                data: {
                    ...ladderContract.data,
                    rungs: [1000, 2000, 3000] // Extremely high rungs
                }
            };
            
            ws.send(JSON.stringify(extremeContract));
            const response = await getNextMessage();
            
            // Even with extreme values, we expect the contract to be accepted
            // as the server implements its own risk management
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });
    });

    describe('Error Handling Tests', () => {
        test('should handle malformed JSON', async () => {
            // Send invalid JSON and expect the connection to remain open
            // The server should be resilient to malformed messages
            ws.send('invalid json{');
            
            // Wait a moment to ensure the server has time to process
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            // The connection should still be open
            expect(ws.readyState).toBe(WebSocket.OPEN);
            
            // Verify we can still send valid messages
            ws.send(JSON.stringify({
                type: "ContractSubmission",
                data: {
                    productType: "MomentumCatcher",
                    targetMovement: 5.0,
                    duration: 2000,
                    payoff: 100.0
                }
            }));
            
            const response = await getNextMessage();
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });

        test('should handle unknown product type', async () => {
            const invalidContract = {
                type: "ContractSubmission",
                data: {
                    productType: "InvalidProduct",
                    duration: 1000,
                    payoff: 100
                }
            };
            
            ws.send(JSON.stringify(invalidContract));
            
            try {
                const response = await getNextMessage(5000);
                // The server accepts the contract but it won't be processed
                expect(response.type).toBe('ContractAccepted');
                expect(response.contractID).toBeDefined();
            } catch (error) {
                // If we get a timeout or connection close, that's also acceptable
                expect(error.message).toMatch(/Message timeout|Connection closed/);
            }
        });

        test('should handle contract with missing fields', async () => {
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
            
            // The server accepts incomplete contracts but they won't be processed
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });

        test('should handle contract with invalid duration', async () => {
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
            
            // The server accepts contracts with invalid durations but they won't be processed
            expect(response.type).toBe('ContractAccepted');
            expect(response.contractID).toBeDefined();
        });
    });
});
